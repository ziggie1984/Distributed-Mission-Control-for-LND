package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	logrus "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
)

// MockGRPCServer is a mock implementation of the GRPCServer interface.
type MockGRPCServer struct {
	mock.Mock
}

func (m *MockGRPCServer) GracefulStop() {
	m.Called()
}

// MockHTTPServer is a mock implementation of the HTTPServer interface.
type MockHTTPServer struct {
	mock.Mock
}

// Shutdown simulates the shutdown of an HTTP server.
//
// This method mocks the behavior of the HTTPServer's Shutdown method. It takes
// a context as an argument and returns an error, which can be predefined in
// the test case using testify's mock functionalities.
func (m *MockHTTPServer) Shutdown(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// TestGracefulShutdown tests the gracefulShutdown function.
func TestGracefulShutdown(t *testing.T) {
	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Create mock servers.
	mockGRPCServer := new(MockGRPCServer)
	mockHTTPServer := new(MockHTTPServer)
	mockPProfServer := new(MockHTTPServer)

	// Setup expectations for the mock servers.
	mockGRPCServer.On("GracefulStop").Return()
	mockHTTPServer.On("Shutdown", mock.Anything).Return(nil)
	mockPProfServer.On("Shutdown", mock.Anything).Return(nil)

	// Create a signal channel.
	sigChan := make(chan os.Signal, 1)

	// Run gracefulShutdown in a separate goroutine.
	go gracefulShutdown(
		sigChan, mockGRPCServer, mockHTTPServer, mockPProfServer,
	)

	// Simulate sending an interrupt signal.
	sigChan <- os.Interrupt

	// Wait a bit to let the gracefulShutdown function complete.
	time.Sleep(100 * time.Millisecond)

	// Assert that all expectations were met.
	mockGRPCServer.AssertExpectations(t)
	mockHTTPServer.AssertExpectations(t)
	mockPProfServer.AssertExpectations(t)
}

// TestGracefulShutdownHTTPError tests gracefulShutdown when the HTTP server
// returns an error during shutdown.
func TestGracefulShutdownHTTPError(t *testing.T) {
	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Create mock servers.
	mockGRPCServer := new(MockGRPCServer)
	mockHTTPServer := new(MockHTTPServer)
	mockPProfServer := new(MockHTTPServer)

	// Setup expectations for the mock servers.
	mockGRPCServer.On("GracefulStop").Return()
	mockHTTPServer.On("Shutdown", mock.Anything).Return(
		fmt.Errorf("HTTP shutdown error"),
	)
	mockPProfServer.On("Shutdown", mock.Anything).Return(nil)

	// Create a signal channel.
	sigChan := make(chan os.Signal, 1)

	// Run gracefulShutdown in a separate goroutine.
	go gracefulShutdown(
		sigChan, mockGRPCServer, mockHTTPServer, mockPProfServer,
	)

	// Simulate sending an interrupt signal.
	sigChan <- os.Interrupt

	// Wait a bit to let the gracefulShutdown function complete.
	time.Sleep(100 * time.Millisecond)

	// Assert that all expectations were met.
	mockGRPCServer.AssertExpectations(t)
	mockHTTPServer.AssertExpectations(t)
	mockPProfServer.AssertExpectations(t)
}

// TestGracefulShutdownPProfError tests gracefulShutdown when the pprof server
// returns an error during shutdown.
func TestGracefulShutdownPProfError(t *testing.T) {
	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Create mock servers.
	mockGRPCServer := new(MockGRPCServer)
	mockHTTPServer := new(MockHTTPServer)
	mockPProfServer := new(MockHTTPServer)

	// Setup expectations for the mock servers.
	mockGRPCServer.On("GracefulStop").Return()
	mockHTTPServer.On("Shutdown", mock.Anything).Return(nil)
	mockPProfServer.On("Shutdown", mock.Anything).Return(
		fmt.Errorf("PProf shutdown error"),
	)

	// Create a signal channel.
	sigChan := make(chan os.Signal, 1)

	// Run gracefulShutdown in a separate goroutine.
	go gracefulShutdown(
		sigChan, mockGRPCServer, mockHTTPServer, mockPProfServer,
	)

	// Simulate sending an interrupt signal.
	sigChan <- os.Interrupt

	// Wait a bit to let the gracefulShutdown function complete.
	time.Sleep(100 * time.Millisecond)

	// Assert that all expectations were met.
	mockGRPCServer.AssertExpectations(t)
	mockHTTPServer.AssertExpectations(t)
	mockPProfServer.AssertExpectations(t)
}
