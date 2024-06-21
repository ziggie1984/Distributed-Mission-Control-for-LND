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
			MaxBatchDelay:   time.Nanosecond,
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
							FailAmtMsat:    100_000,
							SuccessTime:    successTime,
							SuccessAmtSat:  200,
							SuccessAmtMsat: 200_000,
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

		// Case 7: Negative fail amount.
		t.Run("NegativeFailAmount", func(t *testing.T) {
			nodeFrom, nodeTo := generateTestKeys(t)
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						NodeFrom: nodeFrom,
						NodeTo:   nodeTo,
						History: &ecrpc.PairData{
							FailAmtSat:  -100,
							FailAmtMsat: -100 * 1000, // Conversion from sat to msat
						},
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

		// Case 8: Negative success amount.
		t.Run("NegativeSuccessAmount", func(t *testing.T) {
			nodeFrom, nodeTo := generateTestKeys(t)
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						NodeFrom: nodeFrom,
						NodeTo:   nodeTo,
						History: &ecrpc.PairData{
							SuccessAmtSat:  -200,
							SuccessAmtMsat: -200 * 1000, // Conversion from sat to msat
						},
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

		// Case 9: Inconsistent conversion from sat to msat for fail
		// amount.
		t.Run("InconsistentFailAmountConversion", func(t *testing.T) {
			nodeFrom, nodeTo := generateTestKeys(t)
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						NodeFrom: nodeFrom,
						NodeTo:   nodeTo,
						History: &ecrpc.PairData{
							FailAmtSat:  100,
							FailAmtMsat: 50 * 1000, // Inconsistent conversion
						},
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

		// Case 10: Inconsistent conversion from sat to msat for
		// success amount.
		t.Run("InconsistentSuccessAmountConversion", func(t *testing.T) {
			nodeFrom, nodeTo := generateTestKeys(t)
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						NodeFrom: nodeFrom,
						NodeTo:   nodeTo,
						History: &ecrpc.PairData{
							SuccessAmtSat:  200,
							SuccessAmtMsat: 150 * 1000, // Inconsistent conversion
						},
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

		// Case 11: Non-stale history.
		t.Run("NonStaleHistory", func(t *testing.T) {
			// Clear the database.
			err = clearDatabase(db)
			require.NoError(t, err)

			// Register non-stale history data.
			nodeFrom, nodeTo := generateTestKeys(t)
			failTime := time.Now().Add(-5 * time.Minute).Unix()
			successTime := time.Now().Add(-2 * time.Minute).Unix()
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						NodeFrom: nodeFrom,
						NodeTo:   nodeTo,
						History: &ecrpc.PairData{
							FailTime:    failTime,
							SuccessTime: successTime,
						},
					},
				},
			}

			resp, err := server.RegisterMissionControl(
				context.Background(), req,
			)
			require.NoError(t, err)
			require.NotNil(t, resp)

			q_resp, q_err := server.QueryAggregatedMissionControl(
				context.Background(),
				&ecrpc.QueryAggregatedMissionControlRequest{},
			)
			require.NoError(t, q_err)
			require.NotNil(t, q_resp)

			// Check that no data was removed cause there is no
			// stale data.
			require.Len(t, q_resp.Pairs, 1)
		})

		// Case 12: Stale history.
		t.Run("StaleHistory", func(t *testing.T) {
			// Clear the database.
			err = clearDatabase(db)
			require.NoError(t, err)

			// Register stale history data.
			nodeFrom, nodeTo := generateTestKeys(t)
			failTime := time.Now().Add(-15 * time.Minute).Unix()
			successTime := time.Now().Add(-12 * time.Minute).Unix()
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						NodeFrom: nodeFrom,
						NodeTo:   nodeTo,
						History: &ecrpc.PairData{
							FailTime:    failTime,
							SuccessTime: successTime,
						},
					},
				},
			}

			_, err := server.RegisterMissionControl(
				context.Background(), req,
			)
			require.Error(t, err)

			q_resp, q_err := server.QueryAggregatedMissionControl(
				context.Background(),
				&ecrpc.QueryAggregatedMissionControlRequest{},
			)
			require.NoError(t, q_err)

			// Check that there are no data in the db cause all
			// data was stale.
			require.Empty(t, q_resp.Pairs)
		})

		// Case 13: Register new pair with old success and fail times
		// and verify merging works correctly by persisting only the
		// already existing pair recent times.
		t.Run("RegisterNewPairWithOldTimeAndMerge", func(t *testing.T) {
			// Clear the database.
			err = clearDatabase(db)
			require.NoError(t, err)

			// Register a pair with current unix time.
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
							FailAmtMsat:    100_000,
							SuccessTime:    successTime,
							SuccessAmtSat:  200,
							SuccessAmtMsat: 200_000,
						},
					},
				},
			}
			resp, err := server.RegisterMissionControl(
				context.Background(), req,
			)
			require.NoError(t, err)
			require.NotNil(t, resp)

			// Register a new pair with same key but with older
			// success and failed times.
			oSuccessTime := time.Now().Add(-5 * time.Minute).Unix()
			oFailTime := time.Now().Add(-5 * time.Minute).Unix()
			req = &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						NodeFrom: nodeFrom,
						NodeTo:   nodeTo,
						History: &ecrpc.PairData{
							FailTime:       oFailTime,
							FailAmtSat:     100,
							FailAmtMsat:    100_000,
							SuccessTime:    oSuccessTime,
							SuccessAmtSat:  200,
							SuccessAmtMsat: 200_000,
						},
					},
				},
			}
			resp, err = server.RegisterMissionControl(
				context.Background(), req,
			)
			require.NoError(t, err)
			require.NotNil(t, resp)

			q_resp, q_err := server.QueryAggregatedMissionControl(
				context.Background(),
				&ecrpc.QueryAggregatedMissionControlRequest{},
			)
			require.NoError(t, q_err)
			require.NotNil(t, q_resp)
			require.Len(t, q_resp.Pairs, 1)

			// Check the times stored are only the recent times.
			require.Equal(
				t, q_resp.Pairs[0].History.SuccessTime,
				successTime,
			)
			require.Equal(
				t, q_resp.Pairs[0].History.FailTime,
				failTime,
			)
		})

		// Case 14: Register new pair with more recent success and fail
		// times and verify merging works correctly by persisting only
		// the new pair recent times.
		t.Run("RegisterNewPairWithNewTimeAndMerge", func(t *testing.T) {
			// Clear the database.
			err = clearDatabase(db)
			require.NoError(t, err)

			// Register a pair with current unix time.
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
							FailAmtMsat:    100_000,
							SuccessTime:    successTime,
							SuccessAmtSat:  200,
							SuccessAmtMsat: 200_000,
						},
					},
				},
			}
			resp, err := server.RegisterMissionControl(
				context.Background(), req,
			)
			require.NoError(t, err)
			require.NotNil(t, resp)

			// Register a new pair with same key but with recent
			// success and failed times.
			rSuccessTime := time.Now().Add(5 * time.Minute).Unix()
			rFailTime := time.Now().Add(5 * time.Minute).Unix()
			req = &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						NodeFrom: nodeFrom,
						NodeTo:   nodeTo,
						History: &ecrpc.PairData{
							FailTime:       rFailTime,
							FailAmtSat:     100,
							FailAmtMsat:    100_000,
							SuccessTime:    rSuccessTime,
							SuccessAmtSat:  200,
							SuccessAmtMsat: 200_000,
						},
					},
				},
			}
			resp, err = server.RegisterMissionControl(
				context.Background(), req,
			)
			require.NoError(t, err)
			require.NotNil(t, resp)

			q_resp, q_err := server.QueryAggregatedMissionControl(
				context.Background(),
				&ecrpc.QueryAggregatedMissionControlRequest{},
			)
			require.NoError(t, q_err)
			require.NotNil(t, q_resp)
			require.Len(t, q_resp.Pairs, 1)

			// Check the times stored are only the recent times
			// in the new pair registered.
			require.Equal(
				t, q_resp.Pairs[0].History.SuccessTime,
				rSuccessTime,
			)
			require.Equal(
				t, q_resp.Pairs[0].History.FailTime,
				rFailTime,
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
								FailAmtMsat:    100_000,
								SuccessTime:    successTime,
								SuccessAmtSat:  200,
								SuccessAmtMsat: 200_000,
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
						FailAmtMsat:    100_000,
						SuccessTime:    successTime1,
						SuccessAmtSat:  200,
						SuccessAmtMsat: 200_000,
					},
				},
				{
					NodeFrom: nodeFrom,
					NodeTo:   nodeTo,
					History: &ecrpc.PairData{
						FailTime:       failTime2,
						FailAmtSat:     100,
						FailAmtMsat:    100_000,
						SuccessTime:    successTime2,
						SuccessAmtSat:  200,
						SuccessAmtMsat: 200_000,
					},
				},
				{
					NodeFrom: nodeFrom,
					NodeTo:   nodeTo,
					History: &ecrpc.PairData{
						FailTime:       failTime2,
						FailAmtSat:     100,
						FailAmtMsat:    100_000,
						SuccessTime:    successTime2,
						SuccessAmtSat:  200,
						SuccessAmtMsat: 200_000,
					},
				},
			},
		})
		require.NoError(t, err)

		// Mock ticker with a fixed interval.
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		// Create a cancellable context for the cleanup routine.
		cleanupCtx, cleanupCancel := context.WithCancel(
			context.Background(),
		)
		defer cleanupCancel()

		// Start the cleanup routine.
		server.RunCleanupRoutine(cleanupCtx, ticker)

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
						FailAmtMsat:    100_000,
						SuccessTime:    successTime1,
						SuccessAmtSat:  200,
						SuccessAmtMsat: 200_000,
					},
				},
				{
					NodeFrom: nodeFrom,
					NodeTo:   nodeTo,
					History: &ecrpc.PairData{
						FailTime:       failTime2,
						FailAmtSat:     100,
						FailAmtMsat:    100_000,
						SuccessTime:    successTime2,
						SuccessAmtSat:  200,
						SuccessAmtMsat: 200_000,
					},
				},
				{
					NodeFrom: nodeFrom,
					NodeTo:   nodeTo,
					History: &ecrpc.PairData{
						FailTime:       failTime2,
						FailAmtSat:     100,
						FailAmtMsat:    100_000,
						SuccessTime:    successTime2,
						SuccessAmtSat:  200,
						SuccessAmtMsat: 200_000,
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

	t.Run("mergePairData", func(t *testing.T) {
		t.Parallel()

		// Case 1: New data has later success time.
		t.Run("NewDataLaterSuccessTime", func(t *testing.T) {
			existingData := &ecrpc.PairData{
				SuccessTime:    time.Now().Unix(),
				SuccessAmtSat:  100,
				SuccessAmtMsat: 100_000,
				FailTime:       time.Now().Add(-10 * time.Minute).Unix(),
				FailAmtSat:     50,
				FailAmtMsat:    50_000,
			}

			newData := &ecrpc.PairData{
				SuccessTime:    time.Now().Add(5 * time.Minute).Unix(),
				SuccessAmtSat:  200,
				SuccessAmtMsat: 200_000,
				FailTime:       time.Now().Add(-8 * time.Minute).Unix(),
				FailAmtSat:     70,
				FailAmtMsat:    70_000,
			}

			mergePairData(existingData, newData)

			require.Equal(
				t, newData.SuccessTime,
				existingData.SuccessTime,
			)
			require.Equal(
				t, newData.SuccessAmtSat,
				existingData.SuccessAmtSat,
			)
			require.Equal(
				t, newData.SuccessAmtMsat,
				existingData.SuccessAmtMsat,
			)
			require.Equal(
				t, existingData.FailTime,
				existingData.FailTime,
			)
			require.Equal(
				t, existingData.FailAmtSat,
				existingData.FailAmtSat,
			)
			require.Equal(
				t, existingData.FailAmtMsat,
				existingData.FailAmtMsat,
			)
		})

		// Case 2: Existing data has later success time.
		t.Run("ExistingDataLaterSuccessTime", func(t *testing.T) {
			existingData := &ecrpc.PairData{
				SuccessTime:    time.Now().Add(5 * time.Minute).Unix(),
				SuccessAmtSat:  200,
				SuccessAmtMsat: 200_000,
				FailTime:       time.Now().Add(-8 * time.Minute).Unix(),
				FailAmtSat:     70,
				FailAmtMsat:    70_000,
			}

			newData := &ecrpc.PairData{
				SuccessTime:    time.Now().Unix(),
				SuccessAmtSat:  100,
				SuccessAmtMsat: 100_000,
				FailTime:       time.Now().Add(-10 * time.Minute).Unix(),
				FailAmtSat:     50,
				FailAmtMsat:    50_000,
			}

			mergePairData(existingData, newData)

			require.Equal(
				t, existingData.SuccessTime,
				existingData.SuccessTime,
			)
			require.Equal(
				t, existingData.SuccessAmtSat,
				existingData.SuccessAmtSat,
			)
			require.Equal(
				t, existingData.SuccessAmtMsat,
				existingData.SuccessAmtMsat,
			)
			require.Equal(
				t, existingData.FailTime,
				existingData.FailTime,
			)
			require.Equal(
				t, existingData.FailAmtSat,
				existingData.FailAmtSat,
			)
			require.Equal(
				t, existingData.FailAmtMsat,
				existingData.FailAmtMsat,
			)
		})

		// Case 3: New data has later fail time.
		t.Run("NewDataLaterFailTime", func(t *testing.T) {
			existingData := &ecrpc.PairData{
				SuccessTime:    time.Now().Add(-5 * time.Minute).Unix(),
				SuccessAmtSat:  200,
				SuccessAmtMsat: 200_000,
				FailTime:       time.Now().Unix(),
				FailAmtSat:     70,
				FailAmtMsat:    70_000,
			}

			newData := &ecrpc.PairData{
				SuccessTime:    time.Now().Add(-8 * time.Minute).Unix(),
				SuccessAmtSat:  100,
				SuccessAmtMsat: 100_000,
				FailTime:       time.Now().Add(5 * time.Minute).Unix(),
				FailAmtSat:     50,
				FailAmtMsat:    50_000,
			}

			mergePairData(existingData, newData)

			require.Equal(
				t, existingData.SuccessTime,
				existingData.SuccessTime,
			)
			require.Equal(
				t, existingData.SuccessAmtSat,
				existingData.SuccessAmtSat,
			)
			require.Equal(
				t, existingData.SuccessAmtMsat,
				existingData.SuccessAmtMsat,
			)
			require.Equal(
				t, newData.FailTime, existingData.FailTime,
			)
			require.Equal(
				t, newData.FailAmtSat, existingData.FailAmtSat,
			)
			require.Equal(
				t, newData.FailAmtMsat,
				existingData.FailAmtMsat,
			)
		})

		// Case 4: Existing data has later fail time.
		t.Run("ExistingDataLaterFailTime", func(t *testing.T) {
			existingData := &ecrpc.PairData{
				SuccessTime:    time.Now().Add(-8 * time.Minute).Unix(),
				SuccessAmtSat:  100,
				SuccessAmtMsat: 100_000,
				FailTime:       time.Now().Add(5 * time.Minute).Unix(),
				FailAmtSat:     50,
				FailAmtMsat:    50_000,
			}

			newData := &ecrpc.PairData{
				SuccessTime:    time.Now().Add(-5 * time.Minute).Unix(),
				SuccessAmtSat:  200,
				SuccessAmtMsat: 200_000,
				FailTime:       time.Now().Unix(),
				FailAmtSat:     70,
				FailAmtMsat:    70_000,
			}

			mergePairData(existingData, newData)

			require.Equal(
				t, newData.SuccessTime,
				existingData.SuccessTime,
			)
			require.Equal(
				t, newData.SuccessAmtSat,
				existingData.SuccessAmtSat,
			)
			require.Equal(
				t, newData.SuccessAmtMsat,
				existingData.SuccessAmtMsat,
			)
			require.Equal(
				t, existingData.FailTime,
				existingData.FailTime,
			)
			require.Equal(
				t, existingData.FailAmtSat,
				existingData.FailAmtSat,
			)
			require.Equal(
				t, existingData.FailAmtMsat,
				existingData.FailAmtMsat,
			)
		})

		// Case 5: Both new and existing data have the same timestamps.
		t.Run("SameTimestamps", func(t *testing.T) {
			existingData := &ecrpc.PairData{
				SuccessTime:    time.Now().Add(-5 * time.Minute).Unix(),
				SuccessAmtSat:  100,
				SuccessAmtMsat: 100_000,
				FailTime:       time.Now().Unix(),
				FailAmtSat:     50,
				FailAmtMsat:    50_000,
			}

			newData := &ecrpc.PairData{
				SuccessTime:    existingData.SuccessTime,
				SuccessAmtSat:  200,
				SuccessAmtMsat: 200_000,
				FailTime:       existingData.FailTime,
				FailAmtSat:     70,
				FailAmtMsat:    70_000,
			}

			mergePairData(existingData, newData)

			// The function should not modify existingData in this case.
			require.Equal(
				t, existingData.SuccessTime, existingData.SuccessTime,
			)
			require.Equal(
				t, existingData.SuccessAmtSat,
				existingData.SuccessAmtSat,
			)
			require.Equal(
				t, existingData.SuccessAmtMsat,
				existingData.SuccessAmtMsat,
			)
			require.Equal(
				t, existingData.FailTime,
				existingData.FailTime,
			)
			require.Equal(
				t, existingData.FailAmtSat,
				existingData.FailAmtSat,
			)
			require.Equal(
				t, existingData.FailAmtMsat,
				existingData.FailAmtMsat,
			)
		})
	})

	t.Run("isHistoryStale", func(t *testing.T) {
		t.Parallel()

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

			// Make one of the times (failTime) non-stale.
			failTime = time.Now().Add(-5 * time.Minute).Unix()
			history = &ecrpc.PairData{
				FailTime:    failTime,
				SuccessTime: successTime,
			}
			stale = isHistoryStale(
				history, config.Server.HistoryThresholdDuration,
			)
			require.False(t, stale)
		})
	})
}
