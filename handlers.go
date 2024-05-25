package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

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
	config *Config
	db     *bbolt.DB
}

// NewExternalCoordinatorServer creates a new instance of
// ExternalCoordinatorServer.
func NewExternalCoordinatorServer(config *Config,
	db *bbolt.DB) *externalCoordinatorServer {
	return &externalCoordinatorServer{db: db, config: config}
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

	// Sanitize the request data by filtering out pairs with stale history.
	stalePairsRemoved := s.sanitizeRegisterMissionControlRequest(req)

	// Log how many stale history pairs are removed from the request if any.
	if stalePairsRemoved != 0 {
		logrus.Infof("Removed %d stale history pairs",
			stalePairsRemoved)
	}

	// Use Batch over Update to reduce tx commits overhead and database
	// locking, enhancing performance and responsiveness under high write
	// loads.
	err := s.db.Batch(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(DatabaseBucketName))
		for _, pair := range req.Pairs {
			key := append(pair.NodeFrom, pair.NodeTo...)

			data, err := json.Marshal(pair.History)
			if err != nil {
				msg := "failed to unmarshal history data: %v"
				logrus.Errorf(msg, err)
				return status.Errorf(codes.Internal, msg, err)
			}

			if err := b.Put(key, data); err != nil {
				msg := "failed to store data in the bucket: %v"
				logrus.Errorf(msg, err)
				return status.Errorf(codes.Internal, msg, err)
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

	// Construct the registration success message indicating the number of
	// pairs registered.
	successMessage := fmt.Sprintf("Successfully registered %d pairs",
		len(req.Pairs))

	// If there are stale pairs already removed update the registration
	// success message to include the number of pairs removed.
	if stalePairsRemoved > 0 {
		successMessage = fmt.Sprintf("%s and removed %d stale pairs",
			successMessage, stalePairsRemoved)
	}

	// Construct RegisterMissionControlResponse with the success message.
	response := &ecrpc.RegisterMissionControlResponse{
		SuccessMessage: successMessage,
	}

	return response, nil
}

// QueryAggregatedMissionControl queries aggregated mission control data.
func (s *externalCoordinatorServer) QueryAggregatedMissionControl(
	ctx context.Context, req *ecrpc.QueryAggregatedMissionControlRequest) (*ecrpc.QueryAggregatedMissionControlResponse, error) {
	// Log the receipt of the query request.
	logrus.Info("Received QueryAggregatedMissionControl request")

	var pairs []*ecrpc.PairHistory

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(DatabaseBucketName))

		// Pre-allocate memory for the pairs slice based on the
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
				msg := "failed to unmarshal history data: %v"
				logrus.Errorf(msg, err)
				return status.Errorf(codes.Internal, msg, err)
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
		logrus.Infof("Retrieved %d pairs from the database", len(pairs))

		return err
	})
	if err != nil {
		logrus.Errorf("query failed: %v", err)
		return nil, err
	}

	return &ecrpc.QueryAggregatedMissionControlResponse{Pairs: pairs}, nil
}

// RunCleanupRoutine runs a routine to cleanup stale data from the database.
func (s *externalCoordinatorServer) RunCleanupRoutine(ticker *time.Ticker) {
	staleDataCleanupIntervalFormatted := formatDuration(
		s.config.Server.StaleDataCleanupInterval,
	)
	logrus.Infof("Cleanup routine started to remove stale mission "+
		"mission control data from the database on an interval of: "+
		"%s", staleDataCleanupIntervalFormatted)

	// Run the cleanup routine immediately before starting the ticker.
	s.cleanupStaleData()

	// Start a goroutine to handle cleanup routine.
	go func() {
		for range ticker.C {
			// Run the cleanup routine when the ticker ticks.
			s.cleanupStaleData()
		}
	}()
}

// cleanupStaleData cleans up stale mission control data from the database.
func (s *externalCoordinatorServer) cleanupStaleData() {
	logrus.Infof("Running cleanup routine to remove stale mission " +
		"control data from the database...")

	// Initialize a counter to track the number of stale pairs removed.
	stalePairsRemoved := 0

	// Start a read-write transaction to the database.
	err := s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(DatabaseBucketName))

		// Iterate through all key-value pairs in the bucket.
		err := b.ForEach(func(k, v []byte) error {
			history := &ecrpc.PairData{}
			if err := json.Unmarshal(v, history); err != nil {
				msg := "failed to unmarshal history data: %v"
				logrus.Errorf(msg, err)
				return status.Errorf(codes.Internal, msg, err)
			}

			isStale := isHistoryStale(
				history,
				s.config.Server.HistoryThresholdDuration,
			)
			if isStale {
				// If the pair is stale, delete it from the
				// bucket.
				if err := b.Delete(k); err != nil {
					logrus.Errorf("failed to delete "+
						"stale mission control data "+
						"from the bucket: %v", err)
					return nil
				}
				logrus.Debugf("Stale data removed for key: %s",
					hex.EncodeToString(k))

				stalePairsRemoved += 1
			}

			return nil
		})

		if err != nil {
			return fmt.Errorf("error while iterating through "+
				"bucket: %v", err)
		}

		return nil
	})

	if err != nil {
		logrus.Errorf("cleanup routine failed: %v", err)
		return
	}

	logrus.Infof("Cleanup routine completed successfully and %d pairs "+
		"were removed", stalePairsRemoved)
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

	// Flag to track if all pairs are older than the configured threshold.
	allStale := true

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

		isStale := isHistoryStale(
			pair.History, s.config.Server.HistoryThresholdDuration,
		)
		if !isStale {
			// At least one pair is within the threshold.
			allStale = false
		}
	}

	// If all history data pairs are older than the configured threshold,
	// construct an error indicating that none of the pairs can be
	// registered.
	if allStale {
		historyThresholdDurationFormatted := formatDuration(
			s.config.Server.HistoryThresholdDuration,
		)
		return status.Errorf(codes.InvalidArgument, "All history data "+
			"pairs exceed the configured threshold of %s "+
			"and cannot be registered", historyThresholdDurationFormatted,
		)
	}

	return nil
}

// sanitizeRegisterMissionControlRequest sanitizes the RegisterMissionControl
// request by filtering out pairs with stale history and returns the number
// of stale pairs removed.
func (s *externalCoordinatorServer) sanitizeRegisterMissionControlRequest(req *ecrpc.RegisterMissionControlRequest) int {
	// Initialize a counter to track the number of stale pairs removed.
	stalePairsRemoved := 0

	// Iterate through the pairs in reverse order to safely remove elements.
	for i := len(req.Pairs) - 1; i >= 0; i-- {
		pair := req.Pairs[i]

		isStale := isHistoryStale(
			pair.History, s.config.Server.HistoryThresholdDuration,
		)
		if isStale {
			// If the pair is stale, remove it from the slice.
			req.Pairs = append(req.Pairs[:i], req.Pairs[i+1:]...)

			// Increment the counter for stale pairs removed.
			stalePairsRemoved++
		}
	}

	// Return the number of stale pairs removed.
	return stalePairsRemoved
}

// isHistoryStale checks if the history data pair is stale according to the
// configured threshold.
func isHistoryStale(history *ecrpc.PairData, threshold time.Duration) bool {
	// Obtain the most recent UNIX timestamp reflecting temporal
	// locality from the fail_time and success_time fields of the
	// pair's history data. This timestamp will be used to
	// determine whether the pair's history is stale or not.
	recentTimestamp := mostRecentUnixTimestamp(
		history.FailTime, history.SuccessTime,
	)

	// Check if the current history data pair is stale according
	// to the configured threshold duration.
	return time.Unix(recentTimestamp, 0).Before(time.Now().Add(-threshold))
}
