package main

import (
	"context"
	"testing"
	"time"

	btcec "github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/require"
	ecrpc "github.com/ziggie1984/Distributed-Mission-Control-for-LND/ecrpc"
	"go.etcd.io/bbolt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// generateTestKeys generates a pair of test keys for nodeFrom and nodeTo
// identity sec compressed pub keys.
func generateTestKeys(t *testing.T) (nodeFrom, nodeTo []byte) {
	t.Helper()

	// Generate a private key for nodeFrom.
	privKeyFrom, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Serialize the public key of nodeFrom.
	nodeFrom = privKeyFrom.PubKey().SerializeCompressed()

	// Generate a private key for nodeTo.
	privKeyTo, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Serialize the public key of nodeTo.
	nodeTo = privKeyTo.PubKey().SerializeCompressed()

	// Return the serialized public keys.
	return nodeFrom, nodeTo
}

// clearDatabase clears all key-value pairs from the specified bucket.
func clearDatabase(db *bbolt.DB) error {
	return db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(DatabaseBucketName))
		if b == nil {
			// If the bucket does not exist, there is nothing to
			// clear.
			return nil
		}

		// Delete all key-value pairs in the bucket.
		return b.ForEach(func(k, v []byte) error {
			return b.Delete(k)
		})
	})
}

// TestExternalCoordinatorServer tests the RegisterMissionControl and
// QueryAggregatedMissionControl methods of the ExternalCoordinatorServer.
func TestExternalCoordinatorServer(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		Server: ServerConfig{
			HistoryThresholdDuration: 10 * time.Minute,
			StaleDataCleanupInterval: time.Second,
		},
		Database: DatabaseConfig{
			DatabaseDirPath: tempDir,
			DatabaseFile:    "test.db",
			FileLockTimeout: 10 * time.Second,
			MaxBatchDelay:   10 * time.Millisecond,
			MaxBatchSize:    1000,
		},
	}

	db, err := setupDatabase(config)
	if err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	defer cleanupDB(db)

	server := NewExternalCoordinatorServer(config, db)

	t.Run("RegisterMissionControl", func(t *testing.T) {
		// Case 1: Valid request with nodeFrom, nodeTo and history data.
		t.Run("ValidRequest", func(t *testing.T) {
			nodeFrom, nodeTo := generateTestKeys(t)
			failTime := time.Now().Unix()
			successTime := time.Now().Unix()
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						NodeFrom: nodeFrom,
						NodeTo:   nodeTo,
						History: &ecrpc.PairData{
							FailTime:       failTime,
							FailAmtSat:     100,
							FailAmtMsat:    1000,
							SuccessTime:    successTime,
							SuccessAmtSat:  200,
							SuccessAmtMsat: 2000,
						},
					},
				},
			}
			resp, err := server.RegisterMissionControl(
				context.Background(), req,
			)
			require.NoError(t, err)
			require.NotNil(t, resp)
		})

		// Case 2: Nil request.
		t.Run("NilRequest", func(t *testing.T) {
			_, err := server.RegisterMissionControl(
				context.Background(), nil,
			)
			require.Error(t, err)
			require.Equal(
				t, codes.InvalidArgument, status.Code(err),
			)
		})

		// Case 3: Request with empty pairs.
		t.Run("EmptyPairs", func(t *testing.T) {
			req := &ecrpc.RegisterMissionControlRequest{}
			_, err := server.RegisterMissionControl(
				context.Background(), req,
			)
			require.Error(t, err)
			require.Equal(
				t, codes.InvalidArgument, status.Code(err),
			)
		})

		// Case 4: Invalid NodeFrom length.
		t.Run("InvalidNodeFromLength", func(t *testing.T) {
			_, nodeTo := generateTestKeys(t)
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						// Invalid NodeFrom length.
						NodeFrom: []byte{0x01, 0x02},
						NodeTo:   nodeTo,
						History:  &ecrpc.PairData{},
					},
				},
			}
			_, err := server.RegisterMissionControl(
				context.Background(), req,
			)
			require.Error(t, err)
			require.Equal(
				t, codes.InvalidArgument, status.Code(err),
			)
		})

		// Case 5: Invalid NodeTo length.
		t.Run("InvalidNodeToLength", func(t *testing.T) {
			nodeFrom, _ := generateTestKeys(t)
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						NodeFrom: nodeFrom,
						NodeTo:   []byte{0x01, 0x02}, // Invalid length
						History:  &ecrpc.PairData{},
					},
				},
			}
			_, err := server.RegisterMissionControl(
				context.Background(), req,
			)
			require.Error(t, err)
			require.Equal(
				t, codes.InvalidArgument, status.Code(err),
			)
		})

		// Case 6: Nil history data.
		t.Run("NilHistory", func(t *testing.T) {
			nodeFrom, nodeTo := generateTestKeys(t)
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						NodeFrom: nodeFrom,
						NodeTo:   nodeTo,
						History:  nil,
					},
				},
			}
			_, err := server.RegisterMissionControl(
				context.Background(), req,
			)
			require.Error(t, err)
			require.Equal(
				t, codes.InvalidArgument, status.Code(err),
			)
		})
	})

	t.Run("QueryAggregatedMissionControl", func(t *testing.T) {

		// Case 1: Valid request with data in the database.
		t.Run("ValidRequestWithData", func(t *testing.T) {
			err = clearDatabase(db)
			require.NoError(t, err)
			server := NewExternalCoordinatorServer(config, db)
			nodeFrom, nodeTo := generateTestKeys(t)
			failTime := time.Now().Unix()
			successTime := time.Now().Unix()
			_, err = server.RegisterMissionControl(
				context.Background(),
				&ecrpc.RegisterMissionControlRequest{
					Pairs: []*ecrpc.PairHistory{
						{
							NodeFrom: nodeFrom,
							NodeTo:   nodeTo,
							History: &ecrpc.PairData{
								FailTime:       failTime,
								FailAmtSat:     100,
								FailAmtMsat:    1000,
								SuccessTime:    successTime,
								SuccessAmtSat:  200,
								SuccessAmtMsat: 2000,
							},
						},
					},
				})
			require.NoError(t, err)

			resp, err := server.QueryAggregatedMissionControl(
				context.Background(),
				&ecrpc.QueryAggregatedMissionControlRequest{},
			)
			require.NoError(t, err)
			require.NotNil(t, resp)
			// Wait a bit to let the database execute the write
			// batch transactions.
			time.Sleep(1 * time.Second)

			require.Len(t, resp.Pairs, 1)
			require.Equal(t, nodeFrom, resp.Pairs[0].NodeFrom)
			require.Equal(t, nodeTo, resp.Pairs[0].NodeTo)
		})

		// Case 2: Valid request with no data in the database.
		t.Run("ValidRequestWithoutData", func(t *testing.T) {
			err = clearDatabase(db)
			require.NoError(t, err)
			server := NewExternalCoordinatorServer(config, db)
			resp, err := server.QueryAggregatedMissionControl(
				context.Background(),
				&ecrpc.QueryAggregatedMissionControlRequest{},
			)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp.Pairs, 0)
		})
	})

	t.Run("RunCleanupRoutine", func(t *testing.T) {
		// Clear database before running the test.
		err := clearDatabase(db)
		require.NoError(t, err)

		nodeFrom, nodeTo := generateTestKeys(t)

		failTime1 := time.Now().Unix()
		successTime1 := time.Now().Unix()
		failTime2 := time.Now().Add(-15 * time.Minute).Unix()
		successTime2 := time.Now().Add(-12 * time.Minute).Unix()

		// Insert test data into the database.
		_, err = server.RegisterMissionControl(context.Background(), &ecrpc.RegisterMissionControlRequest{
			Pairs: []*ecrpc.PairHistory{
				{
					NodeFrom: nodeFrom,
					NodeTo:   nodeTo,
					History: &ecrpc.PairData{
						FailTime:       failTime1,
						FailAmtSat:     100,
						FailAmtMsat:    1000,
						SuccessTime:    successTime1,
						SuccessAmtSat:  200,
						SuccessAmtMsat: 2000,
					},
				},
				{
					NodeFrom: nodeFrom,
					NodeTo:   nodeTo,
					History: &ecrpc.PairData{
						FailTime:       failTime2,
						FailAmtSat:     100,
						FailAmtMsat:    1000,
						SuccessTime:    successTime2,
						SuccessAmtSat:  200,
						SuccessAmtMsat: 2000,
					},
				},
				{
					NodeFrom: nodeFrom,
					NodeTo:   nodeTo,
					History: &ecrpc.PairData{
						FailTime:       failTime2,
						FailAmtSat:     100,
						FailAmtMsat:    1000,
						SuccessTime:    successTime2,
						SuccessAmtSat:  200,
						SuccessAmtMsat: 2000,
					},
				},
			},
		})
		require.NoError(t, err)

		// Mock ticker with a fixed interval.
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		// Start the cleanup routine.
		server.RunCleanupRoutine(ticker)

		// Wait for some time to ensure the ticker ticks at least once.
		time.Sleep(3 * time.Second)

		// After waiting for the ticker to tick, query the database to
		// check if stale data has been removed.
		resp, err := server.QueryAggregatedMissionControl(
			context.Background(),
			&ecrpc.QueryAggregatedMissionControlRequest{},
		)
		require.NoError(t, err)
		require.NotNil(t, resp)

		// Assert that there is one pair in the response, indicating
		// that all stale data has been removed.
		require.Len(t, resp.Pairs, 1)
	})

	t.Run("cleanupStaleData", func(t *testing.T) {
		// Clear database before running the test.
		err := clearDatabase(db)
		require.NoError(t, err)

		nodeFrom, nodeTo := generateTestKeys(t)

		failTime1 := time.Now().Unix()
		successTime1 := time.Now().Unix()
		failTime2 := time.Now().Add(-15 * time.Minute).Unix()
		successTime2 := time.Now().Add(-12 * time.Minute).Unix()

		// Insert test data into the database.
		_, err = server.RegisterMissionControl(context.Background(), &ecrpc.RegisterMissionControlRequest{
			Pairs: []*ecrpc.PairHistory{
				{
					NodeFrom: nodeFrom,
					NodeTo:   nodeTo,
					History: &ecrpc.PairData{
						FailTime:       failTime1,
						FailAmtSat:     100,
						FailAmtMsat:    1000,
						SuccessTime:    successTime1,
						SuccessAmtSat:  200,
						SuccessAmtMsat: 2000,
					},
				},
				{
					NodeFrom: nodeFrom,
					NodeTo:   nodeTo,
					History: &ecrpc.PairData{
						FailTime:       failTime2,
						FailAmtSat:     100,
						FailAmtMsat:    1000,
						SuccessTime:    successTime2,
						SuccessAmtSat:  200,
						SuccessAmtMsat: 2000,
					},
				},
				{
					NodeFrom: nodeFrom,
					NodeTo:   nodeTo,
					History: &ecrpc.PairData{
						FailTime:       failTime2,
						FailAmtSat:     100,
						FailAmtMsat:    1000,
						SuccessTime:    successTime2,
						SuccessAmtSat:  200,
						SuccessAmtMsat: 2000,
					},
				},
			},
		})
		require.NoError(t, err)

		// Call cleanupStaleData to remove stale data.
		server.cleanupStaleData()

		// Verify that all stale data is removed.
		resp, err := server.QueryAggregatedMissionControl(
			context.Background(),
			&ecrpc.QueryAggregatedMissionControlRequest{},
		)
		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Len(t, resp.Pairs, 1)
	})

	t.Run("sanitizeRegisterMissionControlRequest", func(t *testing.T) {
		// Case 1: No stale pairs.
		t.Run("NoStalePairs", func(t *testing.T) {
			failTime := time.Now().Add(-5 * time.Minute).Unix()
			successTime := time.Now().Add(-2 * time.Minute).Unix()
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						History: &ecrpc.PairData{
							FailTime:    failTime,
							SuccessTime: successTime,
						},
					},
				},
			}
			stalePairsRemoved := server.sanitizeRegisterMissionControlRequest(req)
			require.Equal(t, 0, stalePairsRemoved)
			require.Len(t, req.Pairs, 1)
		})

		// Case 2: All pairs stale.
		t.Run("AllStalePairs", func(t *testing.T) {
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						History: &ecrpc.PairData{
							FailTime:    time.Now().Add(-15 * time.Minute).Unix(),
							SuccessTime: time.Now().Add(-12 * time.Minute).Unix(),
						},
					},
					{
						History: &ecrpc.PairData{
							FailTime:    time.Now().Add(-25 * time.Minute).Unix(),
							SuccessTime: time.Now().Add(-22 * time.Minute).Unix(),
						},
					},
				},
			}
			stalePairsRemoved := server.sanitizeRegisterMissionControlRequest(req)
			require.Equal(t, 2, stalePairsRemoved)
			require.Empty(t, req.Pairs)
		})
	})

	t.Run("isHistoryStale", func(t *testing.T) {
		// Case 1: Non-stale history.
		t.Run("NonStaleHistory", func(t *testing.T) {
			failTime := time.Now().Add(-5 * time.Minute).Unix()
			successTime := time.Now().Add(-2 * time.Minute).Unix()
			history := &ecrpc.PairData{
				FailTime:    failTime,
				SuccessTime: successTime,
			}
			stale := isHistoryStale(
				history, config.Server.HistoryThresholdDuration,
			)
			require.False(t, stale)
		})

		// Case 2: Stale history.
		t.Run("StaleHistory", func(t *testing.T) {
			failTime := time.Now().Add(-15 * time.Minute).Unix()
			successTime := time.Now().Add(-12 * time.Minute).Unix()
			history := &ecrpc.PairData{
				FailTime:    failTime,
				SuccessTime: successTime,
			}
			stale := isHistoryStale(
				history, config.Server.HistoryThresholdDuration,
			)
			require.True(t, stale)
		})
	})
}
