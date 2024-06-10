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

const (
	// PubKeyCompressedSize is the size of a single compressed sec pub key
	// in bytes.
	PubKeyCompressedSize = 33

	// PubKeyCompressedSizeDouble is the size of compressed sec pub keys
	// for both the source and destination nodes in the mission control
	// data pair.
	PubKeyCompressedSizeDouble = PubKeyCompressedSize * 2

	// mSatScale is a value that's used to scale satoshis to
	// milli-satoshis, and the other way around.
	mSatScale int64 = 1000
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

// RegisterMissionControl registers mission control data. It processes a
// RegisterMissionControlRequest to aggregate user-provided pair data with
// existing data in the database, removing stale history pairs and storing the
// aggregated data. This method ensures data consistency and enhances
// performance by utilizing batch operations over individual updates.
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

	// Initialize a map to aggregate mission control data.
	aggregatedData := make(
		map[[PubKeyCompressedSizeDouble]byte]*ecrpc.PairData,
	)

	// Use Batch over Update to reduce tx commits overhead and database
	// locking, enhancing performance and responsiveness under high write
	// loads.
	err := s.db.Batch(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(DatabaseBucketName))

		// Retrieve all data from the database in order to aggregate
		// them later with user registered data.
		err := b.ForEach(func(k, v []byte) error {
			// Unmarshal the pair history data.
			history := &ecrpc.PairData{}
			if err := json.Unmarshal(v, history); err != nil {
				msg := "failed to unmarshal history data: %v"
				logrus.Errorf(msg, err)
				return status.Errorf(codes.Internal, msg, err)
			}

			aggregatedData[[PubKeyCompressedSizeDouble]byte(k)] = history

			return nil
		})
		if err != nil {
			msg := "error while retrieving all data in the " +
				"bucket to aggregate them with user " +
				"registered data: %v"
			logrus.Errorf(msg, err)
			return status.Errorf(codes.Internal, msg, err)
		}

		// Aggregate all data in the database with user registered data.
		for _, pair := range req.Pairs {
			// Aggregate the data based on the key.
			key := [PubKeyCompressedSizeDouble]byte(
				append(pair.NodeFrom, pair.NodeTo...),
			)

			if existingData, ok := aggregatedData[key]; ok {
				// If data for the key exists, merge it with
				// the current data.
				mergePairData(existingData, pair.History)
			} else {
				// If no data exists for the key, set it.
				aggregatedData[key] = pair.History
			}
		}

		// Store the aggregated data.
		for key, value := range aggregatedData {
			// Marshal the pair history data.
			data, err := json.Marshal(value)
			if err != nil {
				msg := "failed to unmarshal history data: %v"
				logrus.Errorf(msg, err)
				return status.Errorf(codes.Internal, msg, err)
			}

			// Store the aggregated data point in the database.
			if err := b.Put([]byte(key[:]), data); err != nil {
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
		msg := "batch operation failed: %v"
		logrus.Errorf(msg, err)
		return nil, status.Errorf(codes.Internal, msg, err)
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

			nodeFrom := k[:PubKeyCompressedSize]
			nodeTo := k[PubKeyCompressedSize:]
			pair := &ecrpc.PairHistory{
				NodeFrom: nodeFrom,
				NodeTo:   nodeTo,
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
		msg := "query failed: %v"
		logrus.Errorf(msg, err)
		return nil, status.Errorf(codes.Internal, msg, err)
	}

	return &ecrpc.QueryAggregatedMissionControlResponse{Pairs: pairs}, nil
}

// RunCleanupRoutine runs a routine to cleanup stale data from the database
// periodically depending on the configured cleanup interval.
func (s *externalCoordinatorServer) RunCleanupRoutine(ctx context.Context,
	ticker *time.Ticker) {
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
		for {
			select {
			case <-ctx.Done():
				// Exit goroutine if the context is canceled.
				return
			case <-ticker.C:
				// Run the cleanup routine when the ticker
				// ticks.
				s.cleanupStaleData()
			}
		}
	}()
}

// cleanupStaleData cleans up stale mission control data from the database.
// It iterates through the database and removes stale data entries.
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
		if len(pair.NodeFrom) != PubKeyCompressedSize {
			return status.Errorf(codes.InvalidArgument, "NodeFrom "+
				"must be exactly %d bytes",
				PubKeyCompressedSize,
			)
		}

		// Validate that NodeTo is exactly 33 bytes i.e compressed sec
		// pub key.
		if len(pair.NodeTo) != PubKeyCompressedSize {
			return status.Errorf(codes.InvalidArgument, "NodeTo "+
				"must be exactly %d bytes",
				PubKeyCompressedSize,
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

		// Validate fail and success amounts are non-negative.
		if pair.History.FailAmtSat < 0 ||
			pair.History.SuccessAmtSat < 0 ||
			pair.History.FailAmtMsat < 0 ||
			pair.History.SuccessAmtMsat < 0 {
			return status.Errorf(
				codes.InvalidArgument, "Fail and success "+
					"amounts must be non-negative",
			)
		}

		// Validate History data is not stale according to configured
		// threshold duration.
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

// mergePairData merges the pair data from two pairs based on the most recent
// timestamp. It does the following:
//   - It updates the success time and amounts if there are more recent history
//     pairs, ensuring that the maximum success amount of the history pair is
//     retained to prevent the success range from shrinking when unnecessary,
//   - It prevents the failure from updating too soon based on the configured
//     MinFailureRelaxInterval value.
//   - It also adjusts the failure range if the success amount goes into the
//     failure range and adjusts the success range if the failure amount goes
//     into the success range.
//
// Parameters:
// - existingData: The existing pair data to merge with.
// - newData: The new pair data to merge with.
func mergePairData(existingData, newData *ecrpc.PairData) {

	if newData.SuccessTime > existingData.SuccessTime {
		// Update success time and amounts if newer, retaining max
		// success amount to avoid shrinking success range
		// unnecessarily.
		existingData.SuccessTime = newData.SuccessTime
		if newData.SuccessAmtMsat > existingData.SuccessAmtMsat {
			existingData.SuccessAmtMsat = newData.SuccessAmtMsat
		}
	}

	if newData.FailTime > existingData.FailTime {
		// Drop result if it would increase the failure amount too soon
		// after a previous failure. This can happen if htlc results
		// come in out of order. This check makes it easier for payment
		// processes to converge to a final state
		newFailureTimestamp := time.Unix(newData.FailTime, 0)
		currentFailureTimestamp := time.Unix(existingData.FailTime, 0)
		failInterval := newFailureTimestamp.Sub(currentFailureTimestamp)
		if newData.FailAmtMsat > existingData.FailAmtMsat &&
			failInterval < MinFailureRelaxInterval {
			logrus.Debugf("Ignoring higher amount failure within "+
				"min failure relaxation interval: "+
				"prev_fail_amt=%v, fail_amt=%v, interval=%v",
				existingData.FailAmtMsat, newData.FailAmtMsat,
				failInterval)
			return
		}

		existingData.FailTime = newData.FailTime
		existingData.FailAmtMsat = newData.FailAmtMsat

		switch {
		// The failure amount is set to zero when the failure is
		// amount-independent, meaning that the attempt would have
		// failed regardless of the amount. This should also reset the
		// success amount to zero.
		case newData.FailAmtMsat == 0:
			existingData.SuccessAmtMsat = 0

		// If the failure range goes into the success range, move the
		// success range down.
		case newData.FailAmtMsat <= existingData.SuccessAmtMsat:
			existingData.SuccessAmtMsat = newData.FailAmtMsat - 1
		}
	}

	// Move the failure range up if the success amount goes into the
	// failure range. We don't want to clear the failure completely
	// because we haven't learnt much for amounts above the current
	// success amount.
	if existingData.FailTime != 0 &&
		newData.SuccessAmtMsat >= existingData.FailAmtMsat {
		existingData.FailAmtMsat = newData.SuccessAmtMsat + 1
	}

	// Update Success and Failure Satoshi amounts based on the
	// millisatoshi unit type, ignoring the fractions of satoshi.
	existingData.SuccessAmtSat = existingData.SuccessAmtMsat / mSatScale
	existingData.FailAmtSat = existingData.FailAmtMsat / mSatScale
}
