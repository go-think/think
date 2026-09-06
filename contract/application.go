package contract

import "github.com/go-think/think/container"

// Application is the contract for the Think application.
type Application interface {
	// GetContainer returns the underlying IoC container for service resolution.
	GetContainer() *container.Container

	// Environment returns true if the current environment matches any of the given environments.
	Environment(envs ...string) bool

	// IsProduction checks if application is running in production.
	IsProduction() bool

	// IsLocal checks if application is running locally.
	IsLocal() bool

	// IsTesting checks if application is running in testing environment.
	IsTesting() bool

	// SetEnvironment sets the application environment.
	SetEnvironment(env string)

	// GetEnvironment returns the current environment name.
	GetEnvironment() string

	// IsDebug checks if the application is running in debug mode.
	IsDebug() bool

	// SetDebug sets the application debug mode.
	SetDebug(debug bool)

	// SetTimezone sets the application default timezone.
	SetTimezone(tz string)

	// GetTimezone returns the application default timezone.
	GetTimezone() string

	// BasePath returns the base path of the application installation.
	BasePath(path ...string) string

	// SetBasePath sets the base path for the application.
	SetBasePath(basePath string) Application

	// ConfigPath returns the path to the configuration directory.
	ConfigPath(path ...string) string

	// StoragePath returns the path to the storage directory.
	StoragePath(path ...string) string

	// PublicPath returns the path to the public directory.
	PublicPath(path ...string) string

	// EnvironmentPath returns the path to the environment file directory.
	EnvironmentPath() string

	// EnvironmentFile returns the environment file name being used.
	EnvironmentFile() string

	// EnvironmentFilePath returns the full path to the environment file.
	EnvironmentFilePath() string
}
