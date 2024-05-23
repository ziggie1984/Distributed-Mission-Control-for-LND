package main

import (
	"context"
	"encoding/json"

	btcec "github.com/btcsuite/btcd/btcec/v2"
	logrus "github.com/sirupsen/logrus"
	ecrpc "github.com/ziggie1984/Distributed-Mission-Control-for-LND/ecrpc"
	bbolt "go.etcd.io/bbolt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// externalCoordinatorServer provides methods to register and query mission
// control data.
type externalCoordinatorServer struct {
	ecrpc.UnimplementedExternalCoordinatorServer
	db *bbolt.DB
}

// NewExternalCoordinatorServer creates a new instance of
// ExternalCoordinatorServer.
func NewExternalCoordinatorServer(db *bbolt.DB) *externalCoordinatorServer {
	return &externalCoordinatorServer{db: db}
}

// RegisterMissionControl registers mission control data.
func (s *externalCoordinatorServer) RegisterMissionControl(ctx context.Context,
	req *ecrpc.RegisterMissionControlRequest) (*ecrpc.RegisterMissionControlResponse, error) {
	// Validate the request data first.
	if err := s.validateRegisterMissionControlRequest(req); err != nil {
		return nil, err
	}

	// Log that there is an incoming request with the number of pairs.
	logrus.Infof("Received RegisterMissionControl request with %d pairs",
		len(req.Pairs))

	// Use Batch over Update to reduce tx commits overhead and database
	// locking, enhancing performance and responsiveness under high write
	// loads.
	err := s.db.Batch(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(DatabaseBucketName))
		for _, pair := range req.Pairs {
			key := append(pair.NodeFrom, pair.NodeTo...)

			data, err := json.Marshal(pair.History)
			if err != nil {
				logrus.Errorf("failed to marshal history "+
					"data: %v", err)

				return status.Errorf(
					codes.Internal, "failed to marshal "+
						"history data: %v", err,
				)
			}

			if err := b.Put(key, data); err != nil {
				logrus.Errorf("failed to store data in the "+
					"bucket: %v", err)

				return status.Errorf(
					codes.Internal, "failed to store data "+
						"in the bucket: %v", err,
				)
			}
		}

		// Log how many pairs are processed and stored.
		logrus.Infof("%d pairs were processed and stored successfully",
			len(req.Pairs))

		return nil
	})

	if err != nil {
		logrus.Errorf("batch operation failed: %v", err)
		return nil, err
	}

	return &ecrpc.RegisterMissionControlResponse{}, nil
}

// QueryAggregatedMissionControl queries aggregated mission control data.
func (s *externalCoordinatorServer) QueryAggregatedMissionControl(
	ctx context.Context, req *ecrpc.QueryAggregatedMissionControlRequest) (*ecrpc.QueryAggregatedMissionControlResponse, error) {
	// Log the receipt of the query request.
	logrus.Info("Received QueryAggregatedMissionControl request")

	var pairs []*ecrpc.PairHistory

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(DatabaseBucketName))

		// Preallocate memory for the pairs slice based on the
		// estimated number of key-value pairs in the bucket. This
		// ensures sufficient capacity to hold all key-value pairs
		// without resizing during iteration.
		//
		// NOTE: The number of estimated keys retrieved may be less or
		// greater than the actual number of keys in the db.
		pairs = make([]*ecrpc.PairHistory, 0, b.Stats().KeyN)
		err := b.ForEach(func(k, v []byte) error {
			history := &ecrpc.PairData{}
			if err := json.Unmarshal(v, history); err != nil {
				logrus.Errorf("Failed to unmarshal history "+
					"data: %v", err)

				return status.Errorf(
					codes.Internal, "failed to unmarshal "+
						"history data: "+err.Error(),
				)
			}

			pair := &ecrpc.PairHistory{
				NodeFrom: k[:33],
				NodeTo:   k[33:],
				History:  history,
			}
			pairs = append(pairs, pair)

			return nil
		})

		// Log the number of pairs retrieved.
		logrus.Infof("Retrieved %d pairs from the database",
			len(pairs))

		return err
	})
	if err != nil {
		logrus.Errorf("Query failed: %v", err)
		return nil, err
	}

	return &ecrpc.QueryAggregatedMissionControlResponse{Pairs: pairs}, nil
}

// validateRegisterMissionControlRequest checks the integrity and correctness
// of the RegisterMissionControlRequest.
func (s *externalCoordinatorServer) validateRegisterMissionControlRequest(req *ecrpc.RegisterMissionControlRequest) error {
	if req == nil {
		return status.Errorf(codes.InvalidArgument, "request cannot "+
			"be nil")
	}

	if len(req.Pairs) == 0 {
		return status.Errorf(codes.InvalidArgument, "request must "+
			"include at least one pair")
	}

	for _, pair := range req.Pairs {
		// Validate that NodeFrom is exactly 33 bytes i.e compressed sec
		// pub key.
		if len(pair.NodeFrom) != 33 {
			return status.Errorf(codes.InvalidArgument, "NodeFrom "+
				"must be exactly 33 bytes",
			)
		}

		// Validate that NodeTo is exactly 33 bytes i.e compressed sec
		// pub key.
		if len(pair.NodeTo) != 33 {
			return status.Errorf(codes.InvalidArgument, "NodeTo "+
				"must be exactly 33 bytes",
			)
		}

		// Validate the NodeFrom public key.
		_, err := btcec.ParsePubKey(pair.NodeFrom)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid "+
				"NodeFrom public key: %v", err,
			)
		}

		// Validate the NodeTo public key.
		_, err = btcec.ParsePubKey(pair.NodeTo)
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "invalid "+
				"NodeTo public key: %v", err,
			)
		}

		// Validate the history data.
		if pair.History == nil {
			return status.Errorf(codes.InvalidArgument, "History "+
				"cannot be nil",
			)
		}
	}
	return nil
}
