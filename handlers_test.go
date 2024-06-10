package main

import (
	"context"
	"testing"
	"time"

	btcec "github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/require"
	ecrpc "github.com/ziggie1984/Distributed-Mission-Control-for-LND/ecrpc"
	"go.etcd.io/bbolt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockQueryAggregatedMissionControlServer is a mock implementation of the
// ecrpc.ExternalCoordinator_QueryAggregatedMissionControlServer interface
// to capture streaming responses in the tests.
type mockQueryAggregatedMissionControlServer struct {
	grpc.ServerStream
	Responses []*ecrpc.QueryAggregatedMissionControlResponse
}

func (m *mockQueryAggregatedMissionControlServer) Send(resp *ecrpc.QueryAggregatedMissionControlResponse) error {
	m.Responses = append(m.Responses, resp)
	return nil
}

func (m *mockQueryAggregatedMissionControlServer) Context() context.Context {
	return context.Background()
}

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

			// Creating a mock stream to capture the responses.
			mockStream := &mockQueryAggregatedMissionControlServer{
				Responses: make([]*ecrpc.QueryAggregatedMissionControlResponse, 0),
			}
			err = server.QueryAggregatedMissionControl(
				&ecrpc.QueryAggregatedMissionControlRequest{},
				mockStream,
			)
			require.NoError(t, err)

			// Check that no data was removed since there is no
			// stale data.
			require.Len(t, mockStream.Responses, 1)
			require.Len(t, mockStream.Responses[0].Pairs, 1)
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

			// Creating a mock stream to capture the responses.
			mockStream := &mockQueryAggregatedMissionControlServer{
				Responses: make([]*ecrpc.QueryAggregatedMissionControlResponse, 0),
			}
			err = server.QueryAggregatedMissionControl(
				&ecrpc.QueryAggregatedMissionControlRequest{},
				mockStream,
			)
			require.NoError(t, err)

			// Check that there are no data in the db since all
			// the data was stale.
			require.Empty(t, mockStream.Responses)
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

			// Creating a mock stream to capture the responses.
			mockStream := &mockQueryAggregatedMissionControlServer{
				Responses: make([]*ecrpc.QueryAggregatedMissionControlResponse, 0),
			}
			err = server.QueryAggregatedMissionControl(
				&ecrpc.QueryAggregatedMissionControlRequest{},
				mockStream,
			)
			require.NoError(t, err)
			require.Len(t, mockStream.Responses, 1)
			require.Len(t, mockStream.Responses[0].Pairs, 1)

			// Check the times stored are only the recent times.
			gotSuccessTime :=
				mockStream.Responses[0].Pairs[0].History.SuccessTime

			gotFailTime :=
				mockStream.Responses[0].Pairs[0].History.FailTime

			require.Equal(t, gotSuccessTime, successTime)
			require.Equal(t, gotFailTime, failTime)
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

			// Creating a mock stream to capture the responses.
			mockStream := &mockQueryAggregatedMissionControlServer{
				Responses: make([]*ecrpc.QueryAggregatedMissionControlResponse, 0),
			}
			err = server.QueryAggregatedMissionControl(
				&ecrpc.QueryAggregatedMissionControlRequest{},
				mockStream,
			)
			require.NoError(t, err)
			require.Len(t, mockStream.Responses, 1)
			require.Len(t, mockStream.Responses[0].Pairs, 1)

			// Check the times stored are only the recent times
			// in the new pair registered.
			gotSuccessTime :=
				mockStream.Responses[0].Pairs[0].History.SuccessTime

			gotFailTime :=
				mockStream.Responses[0].Pairs[0].History.FailTime

			require.Equal(t, gotSuccessTime, rSuccessTime)
			require.Equal(t, gotFailTime, rFailTime)
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

			// Creating a mock stream to capture the responses.
			mockStream := &mockQueryAggregatedMissionControlServer{
				Responses: make([]*ecrpc.QueryAggregatedMissionControlResponse, 0),
			}
			err = server.QueryAggregatedMissionControl(
				&ecrpc.QueryAggregatedMissionControlRequest{},
				mockStream,
			)
			require.NoError(t, err)
			require.Len(t, mockStream.Responses, 1)
			require.Len(t, mockStream.Responses[0].Pairs, 1)
			require.Equal(
				t, nodeFrom,
				mockStream.Responses[0].Pairs[0].NodeFrom,
			)
			require.Equal(
				t, nodeTo,
				mockStream.Responses[0].Pairs[0].NodeTo,
			)
		})

		// Case 2: Valid request with no data in the database.
		t.Run("ValidRequestWithoutData", func(t *testing.T) {
			err = clearDatabase(db)
			require.NoError(t, err)
			server := NewExternalCoordinatorServer(config, db)

			// Creating a mock stream to capture the responses.
			mockStream := &mockQueryAggregatedMissionControlServer{
				Responses: make([]*ecrpc.QueryAggregatedMissionControlResponse, 0),
			}
			err = server.QueryAggregatedMissionControl(
				&ecrpc.QueryAggregatedMissionControlRequest{},
				mockStream,
			)
			require.NoError(t, err)
			require.Len(t, mockStream.Responses, 0)
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

		// Creating a mock stream to capture the responses.
		mockStream := &mockQueryAggregatedMissionControlServer{
			Responses: make([]*ecrpc.QueryAggregatedMissionControlResponse, 0),
		}

		// After waiting for the ticker to tick, query the database to
		// check if stale data has been removed.
		err = server.QueryAggregatedMissionControl(
			&ecrpc.QueryAggregatedMissionControlRequest{},
			mockStream,
		)
		require.NoError(t, err)

		// Assert that there is one pair in the response, indicating
		// that all stale data has been removed.
		require.Len(t, mockStream.Responses, 1)
		require.Len(t, mockStream.Responses[0].Pairs, 1)
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

		// Creating a mock stream to capture the responses.
		mockStream := &mockQueryAggregatedMissionControlServer{
			Responses: make([]*ecrpc.QueryAggregatedMissionControlResponse, 0),
		}

		// Verify that all stale data is removed.
		err = server.QueryAggregatedMissionControl(
			&ecrpc.QueryAggregatedMissionControlRequest{},
			mockStream,
		)
		require.NoError(t, err)
		require.Len(t, mockStream.Responses, 1)
		require.Len(t, mockStream.Responses[0].Pairs, 1)
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

		// Case 1: Update Success Time and Amounts.
		// This test case verifies that if the new data has a more
		// recent success time, the success time and amounts
		// (sat and msat) in the existing data are updated
		// correctly to the new values.
		t.Run("Update Success Time and Amounts", func(t *testing.T) {
			// Initial pair data (existing data).
			existingData := &ecrpc.PairData{
				SuccessTime:    100,
				SuccessAmtSat:  5000,
				SuccessAmtMsat: 5000000,
				FailTime:       90,
				FailAmtSat:     4000,
				FailAmtMsat:    4000000,
			}
			// New pair data to merge.
			newData := &ecrpc.PairData{
				SuccessTime:    110,
				SuccessAmtSat:  6000,
				SuccessAmtMsat: 6000000,
			}

			// Merging new data into existing data.
			mergePairData(existingData, newData)

			// Checking if SuccessTime is updated correctly.
			if existingData.SuccessTime != newData.SuccessTime {
				t.Errorf("Expected SuccessTime %v, got %v", newData.SuccessTime, existingData.SuccessTime)
			}

			// Checking if SuccessAmtSat is updated correctly.
			if existingData.SuccessAmtSat != newData.SuccessAmtSat {
				t.Errorf("Expected SuccessAmtSat %v, got %v", newData.SuccessAmtSat, existingData.SuccessAmtSat)
			}

			// Checking if SuccessAmtMsat is updated correctly.
			if existingData.SuccessAmtMsat !=
				newData.SuccessAmtMsat {
				t.Errorf("Expected SuccessAmtMsat %v, got %v", newData.SuccessAmtMsat, existingData.SuccessAmtMsat)
			}
		})

		// Case 2: Update Failure Time and Amounts.
		// This test case verifies that if the new data has a more
		// recent failure time, the failure time and amounts
		// (sat and msat) in the existing data are updated
		// correctly to the new values.
		t.Run("Update Failure Time and Amounts", func(t *testing.T) {
			// Initial pair data (existing data).
			existingData := &ecrpc.PairData{
				FailTime:    100,
				FailAmtSat:  4000,
				FailAmtMsat: 4000000,
			}

			// New pair data to merge.
			newData := &ecrpc.PairData{
				FailTime:    170,
				FailAmtSat:  5000,
				FailAmtMsat: 5000000,
			}

			// Merging new data into existing data.
			mergePairData(existingData, newData)

			// Checking if FailTime is updated correctly.
			if existingData.FailTime != newData.FailTime {
				t.Errorf("Expected FailTime %v, got %v", newData.FailTime, existingData.FailTime)
			}

			// Checking if FailAmtSat is updated correctly.
			if existingData.FailAmtSat != newData.FailAmtSat {
				t.Errorf("Expected FailAmtSat %v, got %v", newData.FailAmtSat, existingData.FailAmtSat)
			}

			// Checking if FailAmtMsat is updated correctly.
			if existingData.FailAmtMsat != newData.FailAmtMsat {
				t.Errorf("Expected FailAmtMsat %v, got %v", newData.FailAmtMsat, existingData.FailAmtMsat)
			}
		})

		// Case 3: Adjust Success Range.
		// This test case verifies that if the new failure amount
		// goes into the success range, the success range is adjusted
		// correctly to avoid overlap.
		t.Run("Adjust Success Range", func(t *testing.T) {
			// Initial pair data (existing data).
			existingData := &ecrpc.PairData{
				SuccessTime:    100,
				SuccessAmtSat:  7000,
				SuccessAmtMsat: 7000000,
				FailTime:       90,
				FailAmtSat:     8000,
				FailAmtMsat:    8000000,
			}

			// New pair data to merge.
			newData := &ecrpc.PairData{
				FailTime:    110,
				FailAmtSat:  6000,
				FailAmtMsat: 6000000,
			}

			// Merging new data into existing data.
			mergePairData(existingData, newData)

			// Expected values after merge.
			expectedSuccessAmtMsat := newData.FailAmtMsat - 1
			expectedSuccessAmtSat :=
				expectedSuccessAmtMsat / mSatScale

			// Checking if SuccessAmtMsat is adjusted correctly.
			if existingData.SuccessAmtMsat !=
				expectedSuccessAmtMsat {
				t.Errorf("Expected SuccessAmtMsat %v, got %v",
					expectedSuccessAmtMsat,
					existingData.SuccessAmtMsat)
			}

			// Checking if SuccessAmtSat is adjusted correctly.
			if existingData.SuccessAmtSat != expectedSuccessAmtSat {
				t.Errorf("Expected SuccessAmtSat %v, got %v",
					expectedSuccessAmtSat,
					existingData.SuccessAmtSat)
			}
		})

		// Case 4: Adjust Failure Range.
		// This test case verifies that if the new success amount
		// goes into the failure range, the failure range is
		// adjusted correctly to avoid overlap.
		t.Run("Adjust Failure Range", func(t *testing.T) {
			// Initial pair data (existing data).
			existingData := &ecrpc.PairData{
				SuccessTime:    100,
				SuccessAmtSat:  5000,
				SuccessAmtMsat: 5000000,
				FailTime:       90,
				FailAmtSat:     4000,
				FailAmtMsat:    4000000,
			}
			// New pair data to merge.
			newData := &ecrpc.PairData{
				SuccessTime:    110,
				SuccessAmtSat:  6000,
				SuccessAmtMsat: 6000000,
			}

			// Merging new data into existing data.
			mergePairData(existingData, newData)

			// Expected values after merge.
			expectedFailAmtMsat := newData.SuccessAmtMsat + 1
			expectedFailAmtSat :=
				expectedFailAmtMsat / mSatScale

			// Checking if FailAmtMsat is adjusted correctly.
			if existingData.FailAmtMsat !=
				expectedFailAmtMsat {
				t.Errorf("Expected FailAmtMsat %v, got %v",
					expectedFailAmtMsat,
					existingData.FailAmtMsat)
			}

			// Checking if FailAmtSat is adjusted correctly.
			if existingData.FailAmtSat != expectedFailAmtSat {
				t.Errorf("Expected FailAmtSat %v, got %v",
					expectedFailAmtSat,
					existingData.SuccessAmtSat)
			}
		})

		// Case 5: Ignore Higher Failure Amount Within Relaxation
		// Interval.
		//
		// This test case verifies that if a higher failure amount
		// arrives too soon after a previous failure, it is ignored
		// to avoid instability in the failure state.
		t.Run("Ignore Higher Failure Amount Within Relaxation "+
			"Interval", func(t *testing.T) {
			// Initial pair data (existing data).
			earlierFailTime := time.Now().Add(-5 * time.Second)
			existingData := &ecrpc.PairData{
				FailTime:    earlierFailTime.Unix(),
				FailAmtSat:  4000,
				FailAmtMsat: 4000000,
			}
			// New pair data to merge
			newData := &ecrpc.PairData{
				FailTime:    time.Now().Unix(),
				FailAmtSat:  5000,
				FailAmtMsat: 5000000,
			}

			// Merging new data into existing data.
			mergePairData(existingData, newData)

			// Checking if FailAmtSat remains unchanged.
			if existingData.FailAmtSat != 4000 {
				t.Errorf("Expected FailAmtSat to remain %v, got %v", 4000, existingData.FailAmtSat)
			}

			// Checking if FailAmtMsat remains unchanged.
			if existingData.FailAmtMsat != 4000000 {
				t.Errorf("Expected FailAmtMsat to remain %v, got %v", 4000000, existingData.FailAmtMsat)
			}
		})

		// Case 6: Reset Success Amount to Zero for Amount-Independent
		// Failure.
		//
		// This test case verifies that if the new failure amount is
		// zero (indicating an amount-independent failure), the success
		// amounts (sat and msat) in the existing data are reset to
		// zero.
		t.Run("Reset Success Amount to Zero for Amount-Independent "+
			"Failure", func(t *testing.T) {
			// Initial pair data (existing data).
			existingData := &ecrpc.PairData{
				SuccessAmtSat:  5000,
				SuccessAmtMsat: 5000000,
			}

			// New pair data to merge.
			newData := &ecrpc.PairData{
				FailTime:   time.Now().Unix(),
				FailAmtSat: 0,
			}

			// Merging new data into existing data.
			mergePairData(existingData, newData)

			// Checking if SuccessAmtSat is reset to zero.
			if existingData.SuccessAmtSat != 0 {
				t.Errorf("Expected SuccessAmtSat to be reset to 0, got %v", existingData.SuccessAmtSat)
			}

			// Checking if SuccessAmtMsat is reset to zero.
			if existingData.SuccessAmtMsat != 0 {
				t.Errorf("Expected SuccessAmtMsat to be reset to 0, got %v", existingData.SuccessAmtMsat)
			}
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
