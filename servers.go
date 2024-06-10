package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"path/filepath"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	logrus "github.com/sirupsen/logrus"
	ecrpc "github.com/ziggie1984/Distributed-Mission-Control-for-LND/ecrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/encoding/protojson"
)

var (
	// DefaultMarshalOptions defines the marshalling options for JSON
	// output in the gRPC-Gateway. These options ensure the JSON output is
	// consistent mostly with gRPC's default serialization behavior.
	DefaultMarshalOptions = protojson.MarshalOptions{
		// Multiline: false ensures that the JSON output is compact
		// (no pretty-printing). This is useful for reducing the size
		// of the JSON payload and improving performance.
		Multiline: false,

		// EmitUnpopulated: false means that fields with zero values
		// (e.g., empty strings, zero integers) will not be included in
		// the JSON output. This matches the default behavior of gRPC
		// and helps reduce the size of the JSON response.
		EmitUnpopulated: false,
	}
)

// initializeGRPCServer sets up the gRPC server but does not start it.
func initializeGRPCServer(config *Config,
	tlsConfig *tls.Config,
	server *externalCoordinatorServer) (*grpc.Server, net.Listener, error) {
	lis, err := net.Listen(
		"tcp",
		config.Server.GRPCServerHost+config.Server.GRPCServerPort,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen: %v", err)
	}

	// Create the gRPC server with TLS credentials.
	grpcServer := grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)))
	ecrpc.RegisterExternalCoordinatorServer(grpcServer, server)

	return grpcServer, lis, nil
}

// startGRPCServer handles the actual running of the gRPC server.
func startGRPCServer(config *Config, server *grpc.Server,
	lis net.Listener) error {
	logrus.Infof("Starting gRPC server on https://%s%s",
		config.Server.GRPCServerHost, config.Server.GRPCServerPort)

	if err := server.Serve(lis); err != nil {
		return err
	}

	return nil
}

// initializeHTTPServer prepares and returns a configured HTTP server without
// starting it.
func initializeHTTPServer(ctx context.Context,
	tlsConfig *tls.Config,
	config *Config) (*http.Server, error) {
	// Create a new ServeMux to route incoming requests.
	marshalerOption := runtime.WithMarshalerOption(
		runtime.MIMEWildcard, &runtime.JSONPb{
			MarshalOptions: DefaultMarshalOptions,
		},
	)
	mux := runtime.NewServeMux(marshalerOption)

	// Construct the path to the self-signed TLS certificate file.
	tlsCertPath := filepath.Join(
		config.TLS.SelfSignedTLSDirPath,
		config.TLS.SelfSignedTLSCertFile,
	)

	// Read the certificate file.
	certBytes, err := os.ReadFile(tlsCertPath)
	if err != nil {
		return nil, err
	}

	// Create a new certificate pool and add the certificate to it.
	// This certificate pool is used to establish a trusted root CA set,
	// which ensures that the gRPC client verifies the server's certificate
	// during the TLS handshake, thereby securing the communication channel.
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certBytes) {
		return nil, fmt.Errorf("failed to append certificate")
	}

	// Define gRPC dial options with transport credentials using the
	// certificate pool.
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(
			credentials.NewClientTLSFromCert(
				certPool, "",
			),
		),
	}

	err = ecrpc.RegisterExternalCoordinatorHandlerFromEndpoint(
		ctx, mux,
		"localhost"+config.Server.GRPCServerPort,
		opts,
	)
	if err != nil {
		return nil, err
	}

	// Configure HTTP Server settings for the server.
	httpServer := &http.Server{
		Addr: config.Server.RESTServerHost +
			config.Server.RESTServerPort,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	return httpServer, nil
}

// startHTTPServer starts the provided HTTP server for the gRPC REST gateway.
func startHTTPServer(config *Config, httpServer *http.Server) error {
	logrus.Infof("Starting HTTP/1.1 REST server on https://%s%s",
		config.Server.RESTServerHost, DefaultRestServerPort)

	certFile := filepath.Join(
		config.TLS.SelfSignedTLSDirPath,
		config.TLS.SelfSignedTLSCertFile,
	)
	keyFile := filepath.Join(
		config.TLS.SelfSignedTLSDirPath,
		config.TLS.SelfSignedTLSKeyFile,
	)

	err := httpServer.ListenAndServeTLS(certFile, keyFile)
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// initializePProfServer initializes the pprof server but doesn't start it.
func initializePProfServer(config *Config, tlsConfig *tls.Config) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Configure TLS settings for the server.
	pprofServer := &http.Server{
		Addr: config.PProf.PProfServerHost +
			config.PProf.PProfServerPort,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	return pprofServer
}

// startPProfServer starts the pprof server.
func startPProfServer(config *Config, server *http.Server) error {
	logrus.Infof("Starting pprof server on "+
		"https://%s%s", config.PProf.PProfServerHost,
		config.PProf.PProfServerPort)

	certFile := filepath.Join(
		config.TLS.SelfSignedTLSDirPath,
		config.TLS.SelfSignedTLSCertFile,
	)
	keyFile := filepath.Join(
		config.TLS.SelfSignedTLSDirPath,
		config.TLS.SelfSignedTLSKeyFile,
	)
	err := server.ListenAndServeTLS(certFile, keyFile)
	if err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}
