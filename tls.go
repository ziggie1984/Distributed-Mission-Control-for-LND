package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"

	logrus "github.com/sirupsen/logrus"
)

// loadTLSCredentials loads the appropriate TLS credentials based on the
// availability of third-party certificates or falls back to self-signed
// certificates.
//
// It checks for the presence of third-party TLS certificates and, if not found,
// generates and uses local self-signed TLS certificates.
//
// Parameters:
//   - config: A pointer to the Config struct containing TLS configuration
//     settings.
//
// Returns:
//   - A TransportCredentials instance for gRPC if successful, or an error if any
//     step fails.
func loadTLSCredentials(config *Config) (*tls.Config, error) {
	var certFile, keyFile string

	// Check if the third-party TLS certificate and key files are
	// configured.
	if config.TLS.ThirdPartyTLSCertFile != "" &&
		config.TLS.ThirdPartyTLSKeyFile != "" {

		certFile = filepath.Join(
			config.TLS.ThirdPartyTLSDirPath,
			config.TLS.ThirdPartyTLSCertFile,
		)
		keyFile = filepath.Join(
			config.TLS.ThirdPartyTLSDirPath,
			config.TLS.ThirdPartyTLSKeyFile,
		)

		// Check if both third-party files exist.
		err := checkFilesExist(certFile, keyFile)
		if err == nil {
			logrus.Debug("All third-party TLS files found. Using " +
				"third-party TLS certificates.")
		} else {
			logrus.Warn("One or more third-party TLS files are " +
				"missing. Falling back to local TLS " +
				"certificates.")
			certFile, keyFile = "", ""
		}
	} else {
		logrus.Debug("Third-party TLS certificate files not fully " +
			"configured. Using local TLS certificates.")
	}

	// If TLS files are still empty, fall back to local self-signed TLS
	// certificates.
	if certFile == "" && keyFile == "" {
		logrus.Debug("Using local self-signed TLS certificates.")
		certFile = filepath.Join(
			config.TLS.SelfSignedTLSDirPath,
			config.TLS.SelfSignedTLSCertFile,
		)
		keyFile = filepath.Join(
			config.TLS.SelfSignedTLSDirPath,
			config.TLS.SelfSignedTLSKeyFile,
		)
		// Ensure local self-signed TLS certificates exist.
		err := checkAndCreateSelfSignedTLS(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to check/create local "+
				"self-signed TLS certificates: %v", err)
		}
	}

	// Load server's certificate and private key.
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	// Return the TLS credentials for server-side TLS only.
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		ClientAuth:   tls.NoClientCert,
	}, nil
}

// checkAndCreateSelfSignedTLS checks if local self-signed certificates exist and creates them if necessary.
func checkAndCreateSelfSignedTLS(certFile, keyFile string) error {
	err := checkFilesExist(certFile, keyFile)
	if err != nil {
		// If any of them do not exist, re-create them.
		return generateSelfSignedTLS(certFile, keyFile)
	}

	// Load the existing certificate.
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return generateSelfSignedTLS(certFile, keyFile)
	}

	// Check the validity of the existing certificate.
	for _, certData := range cert.Certificate {
		cert, err := x509.ParseCertificate(certData)
		if err != nil {
			return err
		}
		if time.Now().After(cert.NotAfter) {
			logrus.Warning("Self-Signed TLS certificate is " +
				"expired. Creating a new one...")
			return generateSelfSignedTLS(certFile, keyFile)
		}
	}

	return nil
}

// generateSelfSignedTLS generates new self-signed TLS certificates.
//
// It creates a new CA certificate and a server certificate signed by the CA,
// and saves them to the specified file paths.
//
// Parameters:
// - certFile: Path to the server certificate file.
// - keyFile: Path to the server key file.
//
// Returns:
// - An error if the certificate generation fails, or nil if successful.
func generateSelfSignedTLS(certFile, keyFile string) error {
	// Generate a new private key for the server using the P-256 curve.
	serverPriv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	// Valid for one year.
	notBefore := time.Now()
	notAfter := time.Now().Add(365 * 24 * time.Hour)

	// Create a certificate template for the server.
	serverTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Self-Signed Server"},
		},
		NotBefore: notBefore,
		NotAfter:  notAfter,
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
		IsCA:                  true,
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	// Create the server certificate signed by itself (self-signed).
	serverBytes, err := x509.CreateCertificate(
		rand.Reader, &serverTemplate, &serverTemplate,
		&serverPriv.PublicKey, serverPriv,
	)
	if err != nil {
		return err
	}

	// Save the server certificate to the specified file.
	certOut, err := os.Create(certFile)
	if err != nil {
		return err
	}
	defer certOut.Close()

	// Encode the server certificate to PEM format and write it to the file.
	err = pem.Encode(
		certOut, &pem.Block{Type: "CERTIFICATE", Bytes: serverBytes},
	)
	if err != nil {
		return err
	}

	// Save the server private key to the specified file.
	keyOut, err := os.Create(keyFile)
	if err != nil {
		return err
	}
	defer keyOut.Close()

	// Marshal the server private key to DER-encoded format.
	serverPrivBytes, err := x509.MarshalECPrivateKey(serverPriv)
	if err != nil {
		return err
	}

	// Encode the server private key to PEM format and write it to the file.
	err = pem.Encode(
		keyOut,
		&pem.Block{Type: "EC PRIVATE KEY", Bytes: serverPrivBytes},
	)
	if err != nil {
		return err
	}

	return nil
}

// CreateThirdPartyTLSDirIfNotExist checks if the directory for third-party TLS
// certificates exists, and creates it if it does not.
//
// This function ensures that the directory specified in the TLS configuration
// for third-party TLS certificates is present. If the directory does not
// exist, it attempts to create it with the specified permissions.
//
// Parameters:
//   - config: A pointer to the Config struct containing the TLS configuration,
//     including the path to the directory for third-party TLS
//     certificates.
//
// Returns:
//   - An error if the directory creation fails, or nil if the directory already
//     exists or is successfully created.
func CreateThirdPartyTLSDirIfNotExist(config *Config) error {
	// Check if the directory for third-party TLS certificates exists.
	_, err := os.Stat(config.TLS.ThirdPartyTLSDirPath)
	if os.IsNotExist(err) {
		// If the directory does not exist, create it with the
		// specified permissions.
		err := os.Mkdir(
			config.TLS.ThirdPartyTLSDirPath,
			ThirdPartyTLSDirPermissions,
		)
		if err != nil {
			return err
		}
	}

	return nil
}
