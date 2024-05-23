package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	logrus "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// generatePEMData generates PEM encoded data with a custom expiration date.
func generatePEMData(expirationDate time.Time) (certPEM, keyPEM []byte, err error) {
	// Generate a new private key.
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	// Create a template for the certificate.
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore: time.Now(),
		NotAfter:  expirationDate,
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
		},
	}

	// Create a self-signed certificate.
	certDER, err := x509.CreateCertificate(
		rand.Reader, &template, &template, &priv.PublicKey, priv,
	)
	if err != nil {
		return nil, nil, err
	}

	// Encode the certificate to PEM format.
	certPEM = pem.EncodeToMemory(
		&pem.Block{Type: "CERTIFICATE", Bytes: certDER},
	)

	// Marshal the private key to DER format.
	keyDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}

	// Encode the private key to PEM format.
	keyPEM = pem.EncodeToMemory(
		&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER},
	)

	return certPEM, keyPEM, nil
}

// TestLoadTLSCredentials tests the loadTLSCredentials function.
func TestLoadTLSCredentials(t *testing.T) {
	// Define a temporary directory for test TLS files.
	tempDir := t.TempDir()

	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Case 1: Third-party TLS files exist.
	t.Run("Third-party TLS files exist", func(t *testing.T) {
		// Generate valid PEM data for third-party TLS files.
		certPEM, keyPEM, err := generatePEMData(time.Now().Add(365 * 24 * time.Hour))
		assert.NoError(t, err)

		// Create mock third-party TLS files.
		certFile := filepath.Join(tempDir, "third-party-cert.pem")
		keyFile := filepath.Join(tempDir, "third-party-key.pem")
		err = os.WriteFile(certFile, certPEM, 0644)
		assert.NoError(t, err)
		err = os.WriteFile(keyFile, keyPEM, 0644)
		assert.NoError(t, err)

		// Set the config to point to the mock third-party files.
		config := &Config{
			TLS: TLSConfig{
				ThirdPartyTLSDirPath:  tempDir,
				ThirdPartyTLSCertFile: filepath.Base(certFile),
				ThirdPartyTLSKeyFile:  filepath.Base(keyFile),
			},
		}

		creds, err := loadTLSCredentials(config)
		assert.NoError(t, err)
		assert.NotNil(t, creds)
	})

	// Case 2: Falling back to self-signed TLS certificates.
	t.Run("Fallback to self-signed TLS certificates", func(t *testing.T) {
		// Set the config to point to self-signed files.
		config := &Config{
			TLS: TLSConfig{
				SelfSignedTLSDirPath:  tempDir,
				SelfSignedTLSCertFile: "self-signed-cert.pem",
				SelfSignedTLSKeyFile:  "self-signed-key.pem",
			},
		}

		creds, err := loadTLSCredentials(config)
		assert.NoError(t, err)
		assert.NotNil(t, creds)

		// Verify that self-signed files were created.
		assert.FileExists(
			t, filepath.Join(tempDir, "self-signed-cert.pem"),
		)
		assert.FileExists(
			t, filepath.Join(tempDir, "self-signed-key.pem"),
		)
	})

	// Case 3: Self-signed certificates exist but are expired.
	t.Run("Self-signed certificates exist but are expired", func(t *testing.T) {
		// Generate expired PEM data for self-signed certificates.
		expiredDate := time.Now().Add(-365 * 24 * time.Hour)
		certPEM, keyPEM, err := generatePEMData(expiredDate)
		assert.NoError(t, err)

		// Create mock self-signed TLS files.
		certFile := filepath.Join(
			tempDir, "self-signed-cert-expired.pem",
		)
		keyFile := filepath.Join(tempDir, "self-signed-key-expired.pem")
		err = os.WriteFile(certFile, certPEM, 0644)
		assert.NoError(t, err)
		err = os.WriteFile(keyFile, keyPEM, 0644)
		assert.NoError(t, err)

		// Set the config to point to the expired self-signed files.
		config := &Config{
			TLS: TLSConfig{
				SelfSignedTLSDirPath:  tempDir,
				SelfSignedTLSCertFile: filepath.Base(certFile),
				SelfSignedTLSKeyFile:  filepath.Base(keyFile),
			},
		}

		// We expect no error and a returned new tls credentials here
		// because the function generates self-signed certificates if
		// they are expired, lifting this responsibility from the user.
		creds, err := loadTLSCredentials(config)
		assert.NoError(t, err)
		assert.NotNil(t, creds)
	})
}

// TestCheckAndCreateSelfSignedTLS tests the checkAndCreateSelfSignedTLS
// function.
func TestCheckAndCreateSelfSignedTLS(t *testing.T) {
	// Define a temporary directory for test TLS files.
	tempDir := t.TempDir()

	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Case 1: Self-signed TLS files do not exist and are created.
	t.Run("Create self-signed TLS files", func(t *testing.T) {
		certFile := filepath.Join(tempDir, "self-signed-cert.pem")
		keyFile := filepath.Join(tempDir, "self-signed-key.pem")

		err := checkAndCreateSelfSignedTLS(certFile, keyFile)
		assert.NoError(t, err)

		// Verify that self-signed files were created.
		assert.FileExists(t, certFile)
		assert.FileExists(t, keyFile)
	})

	// Case 2: Self-signed TLS files already exist and are valid.
	t.Run("Self-signed TLS files exist and are valid", func(t *testing.T) {
		certFile := filepath.Join(tempDir, "self-signed-cert.pem")
		keyFile := filepath.Join(tempDir, "self-signed-key.pem")

		// Create mock self-signed files.
		err := generateSelfSignedTLS(certFile, keyFile)
		assert.NoError(t, err)

		err = checkAndCreateSelfSignedTLS(certFile, keyFile)
		assert.NoError(t, err)
	})

	// Case 3: Self-signed TLS files exist but are expired.
	t.Run("Self-signed TLS files exist but are expired", func(t *testing.T) {
		certFile := filepath.Join(tempDir, "self-signed-cert-expired.pem")
		keyFile := filepath.Join(tempDir, "self-signed-key-expired.pem")

		// Generate expired self-signed files.
		expiredDate := time.Now().Add(-365 * 24 * time.Hour)
		certPEM, keyPEM, err := generatePEMData(expiredDate)
		assert.NoError(t, err)

		err = os.WriteFile(certFile, certPEM, 0644)
		assert.NoError(t, err)
		err = os.WriteFile(keyFile, keyPEM, 0644)
		assert.NoError(t, err)

		// Check and recreate the self-signed TLS files.
		err = checkAndCreateSelfSignedTLS(certFile, keyFile)
		assert.NoError(t, err)

		// Verify that self-signed files were re-created.
		assert.FileExists(t, certFile)
		assert.FileExists(t, keyFile)

		// Check the expiration date of the recreated certificate.
		certData, err := os.ReadFile(certFile)
		assert.NoError(t, err)

		block, _ := pem.Decode(certData)
		assert.NotNil(t, block)

		cert, err := x509.ParseCertificate(block.Bytes)
		assert.NoError(t, err)
		assert.True(t, cert.NotAfter.After(time.Now()))
	})
}

// TestGenerateSelfSignedTLS tests the generateSelfSignedTLS function.
func TestGenerateSelfSignedTLS(t *testing.T) {
	// Define a temporary directory for test TLS files.
	tempDir := t.TempDir()

	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Case 1: Successfully generate self-signed certificates.
	t.Run("Generate self-signed certificates", func(t *testing.T) {
		certFile := filepath.Join(tempDir, "self-signed-cert.pem")
		keyFile := filepath.Join(tempDir, "self-signed-key.pem")

		err := generateSelfSignedTLS(certFile, keyFile)
		assert.NoError(t, err)

		// Verify that self-signed files were created.
		assert.FileExists(t, certFile)
		assert.FileExists(t, keyFile)
	})
}

// TestCreateThirdPartyTLSDirIfNotExist tests the
// CreateThirdPartyTLSDirIfNotExist function.
func TestCreateThirdPartyTLSDirIfNotExist(t *testing.T) {
	// Define a temporary directory for test TLS directory.
	tempDir := t.TempDir()

	// Suppress log output by setting the output to io.Discard.
	logrus.SetOutput(io.Discard)

	// Case 1: Directory does not exist and is created.
	t.Run("Create third-party TLS directory", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "third-party-tls")

		config := &Config{
			TLS: TLSConfig{
				ThirdPartyTLSDirPath: dirPath,
			},
		}

		err := CreateThirdPartyTLSDirIfNotExist(config)
		assert.NoError(t, err)

		// Verify that the directory was created.
		assert.DirExists(t, dirPath)
	})

	// Case 2: Directory already exists.
	t.Run("Third-party TLS directory already exists", func(t *testing.T) {
		dirPath := filepath.Join(tempDir, "existing-tls-dir")

		// Create the directory.
		err := os.Mkdir(dirPath, 0755)
		assert.NoError(t, err)

		config := &Config{
			TLS: TLSConfig{
				ThirdPartyTLSDirPath: dirPath,
			},
		}

		err = CreateThirdPartyTLSDirIfNotExist(config)
		assert.NoError(t, err)

		// Verify that the directory still exists.
		assert.DirExists(t, dirPath)
	})

	// Case 3: Error creating directory.
	t.Run("Error creating third-party TLS directory", func(t *testing.T) {
		// Use an invalid directory path to force an error.
		dirPath := "/invalid-directory"

		config := &Config{
			TLS: TLSConfig{
				ThirdPartyTLSDirPath: dirPath,
			},
		}

		err := CreateThirdPartyTLSDirIfNotExist(config)
		assert.Error(t, err)
	})
}
