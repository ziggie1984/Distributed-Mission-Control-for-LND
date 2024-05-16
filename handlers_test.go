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

	server := NewExternalCoordinatorServer(db)

	t.Run("RegisterMissionControl", func(t *testing.T) {
		// Case 1: Valid request with nodeFrom, nodeTo and history data.
		t.Run("ValidRequest", func(t *testing.T) {
			nodeFrom, nodeTo := generateTestKeys(t)
			req := &ecrpc.RegisterMissionControlRequest{
				Pairs: []*ecrpc.PairHistory{
					{
						NodeFrom: nodeFrom,
						NodeTo:   nodeTo,
						History: &ecrpc.PairData{
							FailTime:       1,
							FailAmtSat:     100,
							FailAmtMsat:    1000,
							SuccessTime:    2,
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
			server := NewExternalCoordinatorServer(db)
			nodeFrom, nodeTo := generateTestKeys(t)
			_, err = server.RegisterMissionControl(
				context.Background(),
				&ecrpc.RegisterMissionControlRequest{
					Pairs: []*ecrpc.PairHistory{
						{
							NodeFrom: nodeFrom,
							NodeTo:   nodeTo,
							History: &ecrpc.PairData{
								FailTime:       1,
								FailAmtSat:     100,
								FailAmtMsat:    1000,
								SuccessTime:    2,
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

		// Case 1: Valid request with no data in the database.
		t.Run("ValidRequestWithoutData", func(t *testing.T) {
			err = clearDatabase(db)
			require.NoError(t, err)
			server := NewExternalCoordinatorServer(db)
			resp, err := server.QueryAggregatedMissionControl(
				context.Background(),
				&ecrpc.QueryAggregatedMissionControlRequest{},
			)
			require.NoError(t, err)
			require.NotNil(t, resp)
			require.Len(t, resp.Pairs, 0)
		})
	})
}
