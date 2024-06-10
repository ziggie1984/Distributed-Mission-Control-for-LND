package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	logrus "github.com/sirupsen/logrus"
	ecrpc "github.com/ziggie1984/Distributed-Mission-Control-for-LND/ecrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protojson"
)

// getFreePort returns an available TCP port on the local machine for testing
// purposes. It listens on address "127.0.0.1:0" to let the system assign an
// available port.
func getFreePort() (int, error) {
	// Listen on a random port on the local machine.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()

	// Extract and return the port number from the listener's address.
	return l.Addr().(*net.TCPAddr).Port, nil
}

// TestInitializeGRPCServer tests the initialization of the gRPC server.
func TestInitializeGRPCServer(t *testing.T) {
	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Get a free port for the gRPC server.
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get a free port: %v", err)
	}

	// Create a temporary directory for the database. This directory will be
	// automatically deleted at the end of the test.
	tempDir := t.TempDir()

	// Define the configuration for the gRPC server and database.
	config := &Config{
		Server: ServerConfig{
			GRPCServerHost: "localhost",
			GRPCServerPort: fmt.Sprintf(":%d", port),
		},
		Database: DatabaseConfig{
			DatabaseDirPath: tempDir,
			DatabaseFile:    "test.db",
			FileLockTimeout: time.Second,
			MaxBatchDelay:   10 * time.Millisecond,
			MaxBatchSize:    1000,
		},
	}

	db, err := setupDatabase(config)
	if err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	defer cleanupDB(db)

	// Create the external coordinator server.
	server := NewExternalCoordinatorServer(config, db)

	// Initialize the gRPC server with the given configuration and database.
	grpcServer, lis, err := initializeGRPCServer(
		config, &tls.Config{}, server,
	)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if server == nil || lis == nil {
		t.Fatalf("Server or Listener is nil")
	}

	// Stop the server and close the listener.
	lis.Close()
	grpcServer.Stop()
}

// TestInitializeHTTPServer tests the initialization of the HTTP server.
func TestInitializeHTTPServer(t *testing.T) {
	// Get a free port for the gRPC server.
	grpcPort, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get a free gRPC port: %v", err)
	}

	// Get a free port for the HTTP server.
	httpPort, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get a free HTTP port: %v", err)
	}

	// Create a temporary directory for the database. This directory will be
	// automatically deleted at the end of the test.
	tempDir := t.TempDir()

	// Define the configuration for the gRPC and HTTP servers.
	config := &Config{
		TLS: TLSConfig{
			SelfSignedTLSDirPath:  tempDir,
			SelfSignedTLSCertFile: "tls.cert",
			SelfSignedTLSKeyFile:  "tls.key",
		},
		Server: ServerConfig{
			GRPCServerHost: "localhost",
			GRPCServerPort: fmt.Sprintf(":%d", grpcPort),
			RESTServerHost: "localhost",
			RESTServerPort: fmt.Sprintf(":%d", httpPort),
		},
	}

	// Generate the self-signed TLS certificate.
	err = generateSelfSignedTLS(
		filepath.Join(
			config.TLS.SelfSignedTLSDirPath,
			config.TLS.SelfSignedTLSCertFile,
		),
		filepath.Join(
			config.TLS.SelfSignedTLSDirPath,
			config.TLS.SelfSignedTLSKeyFile,
		),
	)
	if err != nil {
		t.Fatalf("Failed to generate a self-signed TLS certificate: "+
			"%v", err)
	}

	ctx := context.Background()

	// Initialize the HTTP server with the given configuration.
	httpServer, err := initializeHTTPServer(ctx, &tls.Config{}, config)
	if err != nil {
		t.Fatalf("Failed to initialize HTTP server: %v", err)
	}

	// Close the HTTP server.
	httpServer.Close()
}

// TestInitializePProfServer tests the initialization of the pprof server.
func TestInitializePProfServer(t *testing.T) {
	// Get a free port for the pprof server.
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get a free port: %v", err)
	}

	// Define the configuration for the pprof server.
	config := &Config{
		PProf: PProfConfig{
			PProfServerHost: "localhost",
			PProfServerPort: fmt.Sprintf(":%d", port),
		},
	}

	// Initialize the pprof server with the given configuration.
	pprofServer := initializePProfServer(config, &tls.Config{})
	if pprofServer == nil {
		t.Fatalf("PProf Server is nil")
	}

	// Close the pprof server.
	pprofServer.Close()
}

// TestStartGRPCServer tests the start of the gRPC server.
func TestStartGRPCServer(t *testing.T) {
	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Get a free port for the gRPC server.
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get a free port: %v", err)
	}

	// Create a temporary directory for the database. The directory will be
	// automatically deleted at the end of the test.
	tempDir := t.TempDir()

	// Define the configuration for the gRPC server and database.
	config := &Config{
		Server: ServerConfig{
			GRPCServerHost:           "localhost",
			GRPCServerPort:           fmt.Sprintf(":%d", port),
			HistoryThresholdDuration: 10 * time.Minute,
			StaleDataCleanupInterval: time.Second,
		},
		TLS: TLSConfig{
			SelfSignedTLSDirPath:  tempDir,
			SelfSignedTLSCertFile: "tls.cert",
			SelfSignedTLSKeyFile:  "tls.key",
		},
		Database: DatabaseConfig{
			DatabaseDirPath: tempDir,
			DatabaseFile:    "test.db",
			FileLockTimeout: time.Second,
			MaxBatchDelay:   10 * time.Millisecond,
			MaxBatchSize:    1000,
		},
	}

	// Use transport credentials for testing.
	tlsConfig, err := loadTLSCredentials(config)
	if err != nil {
		t.Fatalf("Failed to laod tls credentials: %v", err)
	}

	// Set up the database.
	db, err := setupDatabase(config)
	if err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	defer cleanupDB(db)

	// Create the external coordinator server.
	server := NewExternalCoordinatorServer(config, db)

	// Initialize the gRPC server with the given configuration and database.
	grpcServer, lis, err := initializeGRPCServer(config, tlsConfig, server)
	if err != nil {
		t.Fatalf("Failed to initialize gRPC server: %v", err)
	}

	// Create a channel to receive errors from the goroutine.
	errChan := make(chan error, 1)

	// Start the gRPC server in a separate goroutine.
	go func() {
		if err := startGRPCServer(config, grpcServer, lis); err != nil {
			errChan <- fmt.Errorf("Failed to serve gRPC: %v", err)
		}
	}()
	// Ensure the gRPC server is stopped at the end of the test.
	defer grpcServer.Stop()

	tlsCertPath := filepath.Join(
		config.TLS.SelfSignedTLSDirPath,
		config.TLS.SelfSignedTLSCertFile,
	)

	certBytes, err := os.ReadFile(tlsCertPath)
	if err != nil {
		t.Fatalf("Failed to read tls certificate: %v", err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certBytes) {
		t.Fatalf("Failed to append tls certificate: %v", err)
	}

	// Dial the gRPC server.
	ctx := context.Background()
	conn, err := grpc.DialContext(
		ctx,
		fmt.Sprintf(
			"%s%s", config.Server.GRPCServerHost,
			config.Server.GRPCServerPort,
		),
		grpc.WithTransportCredentials(
			credentials.NewClientTLSFromCert(
				certPool, "",
			),
		),
	)
	if err != nil {
		t.Fatalf("Failed to dial gRPC server: %v", err)
	}
	defer conn.Close()

	// Create a client for the ExternalCoordinator service.
	client := ecrpc.NewExternalCoordinatorClient(conn)

	failTime := time.Now().Unix()
	successTime := time.Now().Unix()

	// Register some dummy data.
	nodeFrom, nodeTo := generateTestKeys(t)
	registerReq := &ecrpc.RegisterMissionControlRequest{
		Pairs: []*ecrpc.PairHistory{
			{
				NodeFrom: nodeFrom,
				NodeTo:   nodeTo,
				History: &ecrpc.PairData{
					FailTime:       failTime,
					FailAmtSat:     0,
					FailAmtMsat:    0,
					SuccessTime:    successTime,
					SuccessAmtSat:  0,
					SuccessAmtMsat: 0,
				},
			},
		},
	}
	_, err = client.RegisterMissionControl(ctx, registerReq)
	if err != nil {
		t.Fatalf("RegisterMissionControl request failed: %v", err)
	}

	// Query the registered data.
	req := &ecrpc.QueryAggregatedMissionControlRequest{}
	_, err = client.QueryAggregatedMissionControl(ctx, req)
	if err != nil {
		t.Fatalf("QueryAggregatedMissionControl request failed: %v", err)
	}

	// Check for errors with a timeout.
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatal(err)
		}

	case <-time.After(1 * time.Second):
		// No errors received within the timeout period.
	}

	// Close the error channel.
	close(errChan)
}

// TestStartHTTPServer tests the start of the HTTP server.
func TestStartHTTPServer(t *testing.T) {
	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Get a free port for the gRPC server.
	grpcPort, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get a free gRPC port: %v", err)
	}

	// Get a free port for the HTTP server.
	httpPort, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get a free HTTP port: %v", err)
	}

	// Create a temporary directory for the database that would be
	// automatically deleted at the end of the test.
	tempDir := t.TempDir()

	// Define the configuration for the servers and database.
	config := &Config{
		Server: ServerConfig{
			GRPCServerHost:           "localhost",
			GRPCServerPort:           fmt.Sprintf(":%d", grpcPort),
			RESTServerHost:           "localhost",
			RESTServerPort:           fmt.Sprintf(":%d", httpPort),
			HistoryThresholdDuration: 10 * time.Minute,
			StaleDataCleanupInterval: time.Second,
		},
		TLS: TLSConfig{
			SelfSignedTLSDirPath:  tempDir,
			SelfSignedTLSCertFile: "tls.cert",
			SelfSignedTLSKeyFile:  "tls.key",
		},
		Database: DatabaseConfig{
			DatabaseDirPath: tempDir,
			DatabaseFile:    "test.db",
			FileLockTimeout: 1 * time.Second,
			MaxBatchDelay:   10 * time.Millisecond,
			MaxBatchSize:    1000,
		},
	}

	// Use transport credentials for testing.
	tlsConfig, err := loadTLSCredentials(config)
	if err != nil {
		t.Fatalf("Failed to laod tls credentials: %v", err)
	}

	ctx := context.Background()

	// Set up the test database.
	db, err := setupDatabase(config)
	if err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	defer cleanupDB(db)

	// Create the external coordinator server.
	server := NewExternalCoordinatorServer(config, db)

	// Initialize the gRPC server with the given configuration and database.
	grpcServer, grpcLis, err := initializeGRPCServer(
		config, tlsConfig, server,
	)
	if err != nil {
		t.Fatalf("Failed to initialize gRPC server: %v", err)
	}

	// Create an error channel with a buffer size of two one for the gRPC
	// and the other for the HTTP server.
	errChan := make(chan error, 2)

	// Start the gRPC server in a separate goroutine.
	go func() {
		err := startGRPCServer(config, grpcServer, grpcLis)
		if err != nil {
			errChan <- fmt.Errorf("Failed to serve gRPC: %v", err)
		}
	}()
	// Ensure the gRPC server is stopped at the end of the test.
	defer grpcServer.Stop()

	// Initialize the HTTP server with the given configuration.
	httpServer, err := initializeHTTPServer(ctx, tlsConfig, config)
	if err != nil {
		t.Fatalf("Failed to initialize HTTP server: %v", err)
	}

	// Start the HTTP server in a separate goroutine.
	go func() {
		if err := startHTTPServer(config, httpServer); err != nil {
			errChan <- fmt.Errorf("Failed to serve HTTP REST: %v",
				err)
		}
	}()
	// Ensure the HTTP server is closed at the end of the test.
	defer httpServer.Close()

	tlsCertPath := filepath.Join(
		config.TLS.SelfSignedTLSDirPath,
		config.TLS.SelfSignedTLSCertFile,
	)

	// Read the tls certificate of the server
	certBytes, err := os.ReadFile(tlsCertPath)
	if err != nil {
		t.Fatalf("Failed to read tls certificate: %v", err)
	}

	// Create a new certificate pool and add the TLS certificate to it.
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certBytes) {
		t.Fatalf("Failed to append tls certificate: %v", err)
	}

	// Create a custom Transport that uses the certificate pool.
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: certPool,
		},
	}

	// Create an HTTP client with the custom Transport.
	client := &http.Client{
		Transport: transport,
	}

	// Register some dummy data with the gRPC server.
	conn, err := grpc.DialContext(
		ctx,
		fmt.Sprintf(
			"%s%s", config.Server.GRPCServerHost,
			config.Server.GRPCServerPort,
		),
		grpc.WithTransportCredentials(
			credentials.NewClientTLSFromCert(
				certPool, "",
			),
		),
	)
	if err != nil {
		t.Fatalf("Failed to dial gRPC server: %v", err)
	}
	defer conn.Close()

	failTime := time.Now().Unix()
	successTime := time.Now().Unix()

	clientGRPC := ecrpc.NewExternalCoordinatorClient(conn)
	nodeFrom, nodeTo := generateTestKeys(t)
	registerReq := &ecrpc.RegisterMissionControlRequest{
		Pairs: []*ecrpc.PairHistory{
			{
				NodeFrom: nodeFrom,
				NodeTo:   nodeTo,
				History: &ecrpc.PairData{
					FailTime:       failTime,
					FailAmtSat:     0,
					FailAmtMsat:    0,
					SuccessTime:    successTime,
					SuccessAmtSat:  0,
					SuccessAmtMsat: 0,
				},
			},
		},
	}
	_, err = clientGRPC.RegisterMissionControl(ctx, registerReq)
	if err != nil {
		t.Fatalf("RegisterMissionControl request failed: %v", err)
	}

	// Allow some time for the database to batch the write transaction.
	time.Sleep(1 * time.Second)

	// Send an HTTP GET request to the HTTP server to query the registered
	// data.
	resp, err := client.Get(
		fmt.Sprintf(
			"https://localhost%s/v1/queryaggregatedmissioncontrol", config.Server.RESTServerPort,
		),
	)
	if err != nil {
		t.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check the HTTP response status code.
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("HTTP request failed with status: %v", resp.Status)
	}

	// Read the response body.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read HTTP response body: %v", err)
	}

	// Define a wrapper struct to capture the "result" field which added
	// automatically by grpc-gateway in case of streaming response.
	type WrappedResponse struct {
		Result json.RawMessage `json:"result"`
	}

	// Unmarshal the wrapped response first.
	var wrapped WrappedResponse
	if err := json.Unmarshal(body, &wrapped); err != nil {
		t.Fatalf("Failed to unmarshal wrapped HTTP response: %v", err)
	}

	// Unmarshal the actual response from the "result" field into
	// a QueryAggregatedMissionControlResponse object.
	var response ecrpc.QueryAggregatedMissionControlResponse
	if err := protojson.Unmarshal(wrapped.Result, &response); err != nil {
		t.Fatalf("Failed to unmarshal HTTP response: %v", err)
	}

	// Check if the response contains at least one pair.
	if len(response.Pairs) == 0 {
		t.Fatalf("No pairs found in the response (expected one)")
	}

	// Check for errors with a timeout.
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(1 * time.Second):
		// No errors received within the timeout period.
	}

	// Close the error channel.
	close(errChan)
}

// TestStartPProfServer tests the start of the pprof server.
func TestStartPProfServer(t *testing.T) {
	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Get a free port for the pprof server.
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to get a free port: %v", err)
	}

	// Create a temporary directory for the database. This directory will be
	// automatically deleted at the end of the test.
	tempDir := t.TempDir()

	// Define the configuration for the pprof server.
	config := &Config{
		TLS: TLSConfig{
			SelfSignedTLSDirPath:  tempDir,
			SelfSignedTLSCertFile: "tls.cert",
			SelfSignedTLSKeyFile:  "tls.key",
		},
		PProf: PProfConfig{
			PProfServerHost: "localhost",
			PProfServerPort: fmt.Sprintf(":%d", port),
		},
	}

	// Use transport credentials for testing.
	tlsConfig, err := loadTLSCredentials(config)
	if err != nil {
		t.Fatalf("Failed to laod tls credentials: %v", err)
	}

	tlsCertPath := filepath.Join(
		config.TLS.SelfSignedTLSDirPath,
		config.TLS.SelfSignedTLSCertFile,
	)

	// Read the tls certificate of the server
	certBytes, err := os.ReadFile(tlsCertPath)
	if err != nil {
		t.Fatalf("Failed to read tls certificate: %v", err)
	}

	// Create a new certificate pool and add the TLS certificate to it.
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certBytes) {
		t.Fatalf("Failed to append tls certificate: %v", err)
	}

	// Create a custom Transport that uses the certificate pool.
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: certPool,
		},
	}

	// Create an HTTP client with the custom Transport.
	client := &http.Client{
		Transport: transport,
	}

	// Initialize the pprof server with the given configuration.
	pprofServer := initializePProfServer(config, tlsConfig)
	if pprofServer == nil {
		t.Fatalf("PProf Server is nil")
	}

	// Create a channel to receive errors from the goroutine.
	errChan := make(chan error, 1)

	// Start the pprof server in a separate goroutine.
	go func() {
		if err := startPProfServer(config, pprofServer); err != nil {
			errChan <- fmt.Errorf("Failed to serve pprof: %v", err)
		}
	}()
	// Ensure the pprof server is closed at the end of the test.
	defer pprofServer.Close()

	// Send an HTTP GET request to the pprof server.
	resp, err := client.Get(
		fmt.Sprintf(
			"https://localhost%s/debug/pprof/",
			config.PProf.PProfServerPort,
		),
	)
	if err != nil {
		t.Fatalf("Failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// Check if the HTTP response status code is 200 OK.
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected HTTP status 200, got %v", resp.StatusCode)
	}

	// Check for errors with a timeout.
	select {
	case err := <-errChan:
		if err != nil {
			t.Fatal(err)
		}

	case <-time.After(1 * time.Second):
		// No errors received within the timeout period.
	}

	// Close the error channel.
	close(errChan)
}
