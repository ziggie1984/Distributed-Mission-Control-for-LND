package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Default configuration constants.
const (
	// DefaultAppName is the default app name for the external coordinator
	// service.
	DefaultAppName = "ExternalCoordinator"

	// DefaultConfigFilename is the default filename for the server
	// configuration file.
	DefaultConfigFilename = "ec.conf"

	// DefaultDatabaseDirname is the default directory name for storing
	// database files.
	DefaultDatabaseDirname = "data"

	// DefaultDatabaseFilename is the default filename for the mission
	// control database.
	DefaultDatabaseFilename = "mission_control.db"

	// DefaultTLSCertFilename is the default filename for the TLS
	// self-signed certificate used by the server.
	DefaultTLSCertFilename = "tls.crt"

	// DefaultTLSKeyFilename is the default filename for the TLS key
	// associated with the server's certificate.
	DefaultTLSKeyFilename = "tls.key"

	// DefaultThirdPartyTLSDirname is the default directory name for
	// storing third-party TLS certificates.
	DefaultThirdPartyTLSDirname = "third_party_tls"

	// DefaultLogDirname is the default directory name for storing log
	// files.
	DefaultLogDirname = "logs"

	// DefaultLogFilename is the default filename for the server's
	// operational log.
	DefaultLogFilename = "ec.log"

	// DefaultGrpcServerHost specifies the default host address that the
	// gRPC server will bind to. By default it binds to all network
	// interfaces.
	DefaultGrpcServerHost = "0.0.0.0"

	// DefaultGrpcServerPort specifies the default port that the gRPC
	// server will listen on.
	DefaultGrpcServerPort = ":50050"

	// DefaultRestServerHost specifies the default host address that the
	// REST server will bind to. By default it binds to all network
	// interfaces.
	DefaultRestServerHost = "0.0.0.0"

	// DefaultRestServerPort specifies the default port that the REST
	// server will listen on.
	DefaultRestServerPort = ":8081"

	// DefaultPProfServerHost specifies the default host address that the
	// pprof server will bind to. By default it binds only to the local
	// machine.
	DefaultPProfServerHost = "localhost"

	// DefaultPProfServerPort specifies the default port that the pprof
	// server will listen on.
	DefaultPProfServerPort = ":6060"

	// DefaultLogLevel specifies the default logging level used across the
	// application.
	DefaultLogLevel = "info"

	// DefaultDatabaseFileLockTimeout specifies the default duration to
	// wait for acquiring a file lock on the database before giving up.
	// This timeout helps prevent indefinite blocking on database access
	// in scenarios where multiple processes or threads attempt to access
	// the database concurrently.
	DefaultDatabaseFileLockTimeout = 5 * time.Second

	// DefaultMaxBatchSize specifies the default maximum number of database
	// write operations that can be batched together.
	DefaultMaxBatchSize = 1000

	// DefaultMaxBatchDelay specifies the default maximum delay before a
	// batch of database write operations is committed.
	DefaultMaxBatchDelay = 10 * time.Millisecond

	// DatabaseBucketName specifies the default name of the bucket used
	// within the bbolt database for mission control data.
	DatabaseBucketName = "MissionControl"

	// File and directory permission constants.

	// AppDirPermissions defines the permissions for main application
	// directory. It sets read, write, and execute permissions for the
	// owner, and read and execute permissions for group and others.
	AppDirPermissions = 0755

	// ThirdPartyTLSDirPermissions defines the permissions for third-party
	// TLS directories. It sets read, write, and execute permissions for
	// the owner, and no permissions for group and others.
	ThirdPartyTLSDirPermissions = 0700

	// DatabaseDirPermissions defines the permissions for database
	// directories. It sets read, write, and execute permissions for the
	// owner, and no permissions for group and others.
	DatabaseDirPermissions = 0700

	// LogDirPermissions defines the permissions for log directories. It
	// sets read, write, and execute permissions for the owner, and no
	// permissions for group and others.
	LogDirPermissions = 0700

	// ConfigFilePermissions defines the permissions for configuration
	// files. It sets read and write permissions for the owner, and no
	// permissions for group and others.
	ConfigFilePermissions = 0600

	// DatabaseFilePermissions defines the permissions for database files.
	// It sets read and write permissions for the owner, and no permissions
	// for group and others.
	DatabaseFilePermissions = 0600

	// LogFilePermissions defines the permissions for log files. It sets
	// read and write permissions for the owner, and no permissions for
	// group and others.
	LogFilePermissions = 0600
)

// Config holds the overall configuration values for the server.
type Config struct {
	Server   ServerConfig   `mapstructure:"server" description:"Configuration settings related to server endpoints, including both gRPC and REST servers."`
	PProf    PProfConfig    `mapstructure:"pprof" description:"Configuration for the pprof server used for monitoring and profiling the application."`
	TLS      TLSConfig      `mapstructure:"tls" description:"Configuration related to Transport Layer Security (TLS), including settings for both self-signed and third-party certificates."`
	Database DatabaseConfig `mapstructure:"database" description:"Database configuration settings, including the path, filename, and operational parameters like timeouts and batch sizes."`
	Log      LogConfig      `mapstructure:"log" description:"Logging configuration, specifying the path, file, and level of logging detail."`
}

// ServerConfig holds the server configuration values.
type ServerConfig struct {
	GRPCServerHost string `mapstructure:"grpc_server_host" description:"The host address for the gRPC server. Specify the IP address or hostname that the gRPC server will bind to. Default is '0.0.0.0', which represents all available network interfaces."`
	GRPCServerPort string `mapstructure:"grpc_server_port" description:"The port number for the gRPC server. This is the port on which the gRPC server will listen for incoming connections."`
	RESTServerHost string `mapstructure:"rest_server_host" description:"The host address for the RESTful server interface provided via gRPC Gateway. It determines the network address the HTTP server binds to. Default is '0.0.0.0, which represents all available network interfaces."`
	RESTServerPort string `mapstructure:"rest_server_port" description:"The port number for the RESTful HTTP server. This port will be used for handling HTTP requests that are translated into gRPC calls."`
}

// PProfConfig holds the pprof configuration values.
type PProfConfig struct {
	PProfServerHost string `mapstructure:"pprof_server_host" description:"The host address for the pprof server, used for profiling and monitoring the application. By default The server only binds to the localhost."`
	PProfServerPort string `mapstructure:"pprof_server_port" description:"The port number on which the pprof server will listen. pprof provides runtime profiling data via a web interface."`
}

// TLSConfig holds the TLS configuration values.
type TLSConfig struct {
	SelfSignedTLSDirPath  string `mapstructure:"self_signed_tls_dir_path" description:"Directory path where self-signed TLS certificates are stored. This path is typically used when no third-party certificates are provided."`
	SelfSignedTLSCertFile string `mapstructure:"self_signed_tls_cert_file" description:"Filename of the self-signed TLS certificate used by the server. It should be located within the directory specified in 'self_signed_tls_dir_path'."`
	SelfSignedTLSKeyFile  string `mapstructure:"self_signed_tls_key_file" description:"Filename of the private key corresponding to the self-signed TLS certificate."`
	ThirdPartyTLSDirPath  string `mapstructure:"third_party_tls_dir_path" description:"Directory path that stores third-party TLS certificates, if available. This is used when certificates are provided by an external certificate authority."`
	ThirdPartyTLSCertFile string `mapstructure:"third_party_tls_cert_file" description:"Filename of the third-party TLS certificate. This certificate is used if available, falling back to self-signed if not."`
	ThirdPartyTLSKeyFile  string `mapstructure:"third_party_tls_key_file" description:"Filename of the private key for the third-party TLS certificate."`
}

// DatabaseConfig holds the database configuration values.
type DatabaseConfig struct {
	DatabaseDirPath string        `mapstructure:"database_dir_path" description:"The filesystem path to the directory where the database file is stored. Ensures all database operations are confined to this directory."`
	DatabaseFile    string        `mapstructure:"database_file" description:"The filename of the database where mission control data is persisted."`
	FileLockTimeout time.Duration `mapstructure:"file_lock_timeout" description:"The maximum time to wait for acquiring a database file lock before the operation times out. This setting is crucial for preventing deadlocks and ensuring smooth database operation under concurrent access conditions."`
	MaxBatchSize    int           `mapstructure:"max_batch_size" description:"The maximum number of database operations to batch together. This can improve performance by reducing the number of writes to disk."`
	MaxBatchDelay   time.Duration `mapstructure:"max_batch_delay" description:"The maximum delay before a batch of database operations is committed. Balancing this delay can help in optimizing the responsiveness and throughput of the database."`
}

// LogConfig holds the log configuration values.
type LogConfig struct {
	LogDirPath string `mapstructure:"log_dir_path" description:"Directory where log files are stored. Centralizes logging output to this location for easier management and review."`
	LogFile    string `mapstructure:"log_file" description:"Filename for the log file where runtime information and errors are recorded."`
	LogLevel   string `mapstructure:"log_level" description:"The level of logging detail. Options are 'fatal', 'error', 'warn', 'warning', 'info', 'debug'. Lower levels provide more detailed output for troubleshooting and higher levels provide condensed output for general monitoring."`
}

// DefaultConfig returns a Config initialized with default values.
func DefaultConfig() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("Failed to get user home "+
			"directory: %v", err)
	}
	appPath := AppPath(runtime.GOOS, homeDir)
	return Config{
		Server: ServerConfig{
			GRPCServerHost: DefaultGrpcServerHost,
			GRPCServerPort: DefaultGrpcServerPort,
			RESTServerHost: DefaultRestServerHost,
			RESTServerPort: DefaultRestServerPort,
		},
		PProf: PProfConfig{
			PProfServerHost: DefaultPProfServerHost,
			PProfServerPort: DefaultPProfServerPort,
		},
		TLS: TLSConfig{
			SelfSignedTLSDirPath:  appPath,
			SelfSignedTLSCertFile: DefaultTLSCertFilename,
			SelfSignedTLSKeyFile:  DefaultTLSKeyFilename,
			ThirdPartyTLSDirPath: filepath.Join(appPath,
				DefaultThirdPartyTLSDirname),
		},
		Database: DatabaseConfig{
			DatabaseDirPath: filepath.Join(appPath,
				DefaultDatabaseDirname),
			DatabaseFile:    DefaultDatabaseFilename,
			FileLockTimeout: DefaultDatabaseFileLockTimeout,
			MaxBatchSize:    DefaultMaxBatchSize,
			MaxBatchDelay:   DefaultMaxBatchDelay,
		},
		Log: LogConfig{
			LogDirPath: filepath.Join(appPath, DefaultLogDirname),
			LogFile:    DefaultLogFilename,
			LogLevel:   DefaultLogLevel,
		},
	}, nil
}

// initConfig initializes the configuration from a file or creates a new one
// with defaults. It reads configuration values from a .conf file located in the
// application directory or creates a new file with default settings if the
// file does not exist.
func initConfig(path, configFileName string) (*Config, error) {
	// Construct the path of the config file.
	configFilePath := filepath.Join(path, configFileName)

	// Check if the configuration file exists, if not create one with
	// default settings.
	defaultConfig, err := DefaultConfig()
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// os.O_CREATE | os.O_WRONLY | os.O_TRUNC is equivalent to
		// behavior of os.Create.
		configFile, err := os.OpenFile(
			configFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, ConfigFilePermissions,
		)
		if err != nil {
			return nil, err
		}
		defer configFile.Close()

		err = writeConfigSection(
			configFile, reflect.ValueOf(defaultConfig),
			reflect.TypeOf(defaultConfig), "",
		)
		if err != nil {
			return nil, err
		}
	}

	// Set the configuration file path and format to Viper. Viper is used
	// here as the configuration management tool.
	viper.SetConfigFile(configFilePath)
	viper.SetConfigType("ini")

	// Attempt to read the configuration file content using Viper.
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	// Unmarshal the read configuration data into the Config struct defined
	// in the application.
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %v", err)
	}

	// Return loaded configuration and a nil error on success.
	return &config, nil
}

// writeConfigSection writes a configuration section to the provided file.
// This function is recursive for nested structs in the configuration structure.
func writeConfigSection(w io.Writer, val reflect.Value, typ reflect.Type,
	prefix string) error {
	// Check if the writer is nil.
	if w == nil {
		return fmt.Errorf("writer cannot be nil")
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		iniTag := fieldType.Tag.Get("mapstructure")
		descTag := fieldType.Tag.Get("description")

		// Format description text to fit within 80 characters.
		wrappedDesc := wrapText(descTag, 80)

		// Check if the field is a struct which would represent a
		// nested configuration section.
		if field.Kind() == reflect.Struct {
			// Construct the section name using the 'mapstructure'
			// tag from the struct field. This tag defines the name
			// in the configuration file.
			sectionName := iniTag

			// If there's a prefix, prepend it to create a nested
			// section name. This is used for deeper nested
			// structures to maintain the hierarchy in the INI file.
			if prefix != "" {
				sectionName = fmt.Sprintf("%s.%s",
					prefix, iniTag)
			}

			// If a description tag is present, write it as a
			// comment above the section. This helps document the
			// purpose of the section directly in the configuration
			// file.
			if descTag != "" {
				_, err := fmt.Fprintf(w, "; %s\n", wrappedDesc)
				if err != nil {
					return err
				}
			}

			// Write the section name in brackets to denote the
			// start of a new section.
			_, err := fmt.Fprintf(w, "[%s]\n", sectionName)
			if err != nil {
				return err
			}

			// Recursively call writeConfigSection to process
			// fields of the struct. This allows the function
			// to handle structures of arbitrary depth.
			err = writeConfigSection(
				w, field, fieldType.Type, sectionName,
			)
			if err != nil {
				return err
			}
		} else {
			// For non-struct fields, write them as individual
			// key-value pairs.
			// If a description exists, it's written as a comment
			// above the key-value pair.
			if descTag != "" {
				_, err := fmt.Fprintf(
					w, "; %s\n", wrappedDesc,
				)
				if err != nil {
					return err
				}
			}

			// Write the key-value pair in the format 'key = value'.
			// The key is taken from the 'mapstructure' tag which
			// should match the structure's field name.
			_, err := fmt.Fprintf(
				w, "%s = %v\n\n", iniTag, field.Interface(),
			)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// wrapText wraps the provided text at the specified line width for better
// readability in configuration files.
func wrapText(text string, lineWidth int) string {
	// Return the text as is if line width is zero or negative.
	if lineWidth <= 0 {
		return text
	}

	// Split the text into words, trimming any leading or trailing
	// whitespace.
	words := strings.Fields(strings.TrimSpace(text))
	if len(words) == 0 {
		return text
	}

	// Start with the first word.
	wrapped := words[0]

	// Calculate the remaining space in the line after the first word.
	spaceLeft := lineWidth - len(wrapped)
	for _, word := range words[1:] {
		// If adding the next word exceeds the lineWidth,
		// wrap to a new line.
		if len(word)+1 > spaceLeft {
			// Start a new line with the word, preceded by '; '.
			wrapped += "\n; " + word
			// Reset spaceLeft for the new line.
			spaceLeft = lineWidth - len(word)
		} else {
			// If the word fits in the remaining line, add it with
			// a preceding space.
			wrapped += " " + word
			// Update spaceLeft after adding the word.
			spaceLeft -= (1 + len(word))
		}
	}

	// Return the wrapped text formatted for configuration file comments.
	return wrapped
}
