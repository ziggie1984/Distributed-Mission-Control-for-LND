package main

import (
	"context"
	"os"
	"time"

	logrus "github.com/sirupsen/logrus"
)

// GRPCServer defines an interface for a gRPC server.
type GRPCServer interface {
	GracefulStop()
}

// HTTPServer defines an interface for an HTTP server.
type HTTPServer interface {
	Shutdown(ctx context.Context) error
}

// gracefulShutdown handles graceful shutdown of the servers.
func gracefulShutdown(sigChan chan os.Signal, grpcServer GRPCServer,
	httpServer HTTPServer, pprofServer HTTPServer) {
	// Block until a signal is received.
	<-sigChan
	logrus.Info("Shutting down servers...")

	// Graceful shutdown the gRPC server.
	grpcServer.GracefulStop()
	logrus.Info("gRPC server has been stopped.")

	// Graceful shutdown the HTTP server.
	if err := httpServer.Shutdown(context.Background()); err != nil {
		logrus.Errorf("HTTP server shutdown error: %v", err)
	} else {
		logrus.Info("HTTP server has been stopped.")
	}

	// Create context timeout with 5 seconds for pprof server to shutdown.
	pprofCtx, pprofCancel := context.WithTimeout(
		context.Background(), 5*time.Second,
	)
	defer pprofCancel()

	// Graceful shutdown the pprof server.
	if err := pprofServer.Shutdown(pprofCtx); err != nil {
		logrus.Errorf("PProf server shutdown error: %v", err)
	} else {
		logrus.Info("PProf server has been stopped.")
	}

	logrus.Info("Exited gracefully")
}
