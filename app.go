package think

import (
	"net/http"

	"github.com/go-think/flow"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-think/think/config"
	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
	"github.com/go-think/think/facades"
	"github.com/go-think/think/provider"
	"github.com/go-think/think/support"
	"github.com/go-think/think/support/env"
)

// Application the ThinkGo Application and IoC Container
type Application struct {
	*container.Container

	basePath        string
	environmentPath string // The path to the environment file directory
	environmentFile string // The environment file to load (default: ".env")
	env             string
	debug           bool
	timezone        string
	providers       []support.ServiceProvider
	bootstrapped    bool
	booted          bool
	server          *http.Server
}

// New returns a new Application instance with BasePath and core base bindings registered.
func New(basePath ...string) *Application {
	bp := ""
	if len(basePath) > 0 && basePath[0] != "" {
		bp = basePath[0]
	} else {
		// Default to current working directory
		bp, _ = os.Getwd()
	}

	app := &Application{
		Container:       container.New(),
		environmentFile: ".env",
		env:             "production",
		providers:       make([]support.ServiceProvider, 0),
	}

	// 1. Set base path
	if bp != "" {
		app.SetBasePath(bp)
	}

	// 2. Register base container bindings
	app.registerBaseBindings()

	// 3. Register base service providers
	app.registerBaseServiceProviders()

	return app
}

// UseEnvironmentPath sets the directory for the environment file.
func (a *Application) UseEnvironmentPath(path string) *Application {
	a.environmentPath = path
	return a
}

// EnvironmentPath returns the path to the environment file directory.
func (a *Application) EnvironmentPath() string {
	if a.environmentPath != "" {
		return a.environmentPath
	}
	return a.basePath
}

// LoadEnvironmentFrom sets the environment file to load during bootstrapping.
func (a *Application) LoadEnvironmentFrom(file string) *Application {
	if filepath.IsAbs(file) {
		a.environmentPath = filepath.Dir(file)
		a.environmentFile = filepath.Base(file)
	} else {
		a.environmentFile = file
	}
	return a
}

// EnvironmentFile returns the environment file name being used.
func (a *Application) EnvironmentFile() string {
	if a.environmentFile == "" {
		return ".env"
	}
	return a.environmentFile
}

// EnvironmentFilePath returns the full path to the environment file.
func (a *Application) EnvironmentFilePath() string {
	if filepath.IsAbs(a.EnvironmentFile()) {
		return a.EnvironmentFile()
	}
	return filepath.Join(a.EnvironmentPath(), a.EnvironmentFile())
}

// checkForSpecificEnvironmentFile detects if a custom environment file matching APP_ENV exists.
func (a *Application) checkForSpecificEnvironmentFile() {
	// If the user has already explicitly specified a custom environment file (other than default .env), respect it
	if a.environmentFile != "" && a.environmentFile != ".env" {
		return
	}

	targetEnv := ""

	// 1. Check command line arguments for --env=xxx or --env xxx
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		if strings.HasPrefix(arg, "--env=") {
			targetEnv = strings.TrimPrefix(arg, "--env=")
			break
		}
		if arg == "--env" && i+1 < len(os.Args) {
			targetEnv = os.Args[i+1]
			break
		}
	}

	// 2. Check system environment variable APP_ENV
	if targetEnv == "" {
		targetEnv = os.Getenv("APP_ENV")
	}

	if targetEnv == "" {
		return
	}

	// Check if .env.{environment} exists in environmentPath
	specificFile := a.EnvironmentFile() + "." + targetEnv
	fullPath := filepath.Join(a.EnvironmentPath(), specificFile)
	if _, err := os.Stat(fullPath); err == nil {
		a.LoadEnvironmentFrom(specificFile)
	}
}

// bootstrapEnvironmentVariables loads and binds the environment variables from disk.
func (a *Application) bootstrapEnvironmentVariables() {
	// 1. Automatically detect custom environment file (.env.{environment})
	a.checkForSpecificEnvironmentFile()

	envFile := a.EnvironmentFilePath()
	_ = env.LoadWithPriority(envFile)

	// Sync environment if configured
	if envVal := env.Get("APP_ENV"); envVal != "" {
		a.SetEnvironment(envVal)
	}
}

// bootstrapConfiguration mounts and configures the configuration repository.
func (a *Application) bootstrapConfiguration() {
	var repository contract.Config
	if cfg := a.Make[contract.Config](); cfg != nil {
		repository = cfg
	}

	if repository == nil {
		repository = config.Default()
		a.Instance[contract.Config](repository)
	}

	// 1. Detect and set application environment from config
	if envVal := repository.GetString("app.env"); envVal != "" {
		a.SetEnvironment(envVal)
	}

	// 2. Set application debug mode from config
	if repository.Has("app.debug") {
		a.SetDebug(repository.GetBool("app.debug"))
	}

	// 3. Set default application timezone from config
	if tz := repository.GetString("app.timezone"); tz != "" {
		a.SetTimezone(tz)
	}
}

// routeDump provides the console route:list command with the route table.
func (a *Application) routeDump() func() (string, bool) {
	return func() (string, bool) {
		router := a.Make[flow.Router]()
		if router == nil {
			return "", false
		}
		router.Register()
		dump := string(router.Dump())
		return dump, dump != ""
	}
}

// HasBeenBootstrapped checks if the application has been bootstrapped.
func (a *Application) HasBeenBootstrapped() bool {
	return a.bootstrapped
}

// registerBaseBindings registers the basic bindings into the container.
func (a *Application) registerBaseBindings() {
	// Set global facades app
	facades.App = a

	// Bind application & container singletons
	a.Instance[*Application](a)
	a.Instance[contract.Application](a)
	a.Alias[contract.Application]("app")

	// Early base config repository
	baseConfig := config.Default()
	a.Instance[contract.Config](baseConfig)
	a.Alias[contract.Config]("config")
}

// registerBaseServiceProviders registers core base infrastructure service providers.
func (a *Application) registerBaseServiceProviders() {
	a.Register(&provider.EventServiceProvider{})
	a.Register(&provider.LogServiceProvider{})
	a.Register(&provider.RoutingServiceProvider{})
	a.Register(&provider.CacheServiceProvider{})
}

// bindPathsInContainer binds all application path singletons in the container.
func (a *Application) bindPathsInContainer() {
	a.Instance[string](a.BasePath(), "path.base")
	a.Instance[string](a.ConfigPath(), "path.config")
	a.Instance[string](a.StoragePath(), "path.storage")
	a.Instance[string](a.PublicPath(), "path.public")
}

// BasePath returns the base path of the application installation.
func (a *Application) BasePath(path ...string) string {
	if len(path) > 0 && path[0] != "" {
		return filepath.Join(a.basePath, filepath.Clean(path[0]))
	}
	return a.basePath
}

// SetBasePath sets the base path for the application.
func (a *Application) SetBasePath(basePath string) contract.Application {
	a.basePath = filepath.Clean(strings.TrimRight(basePath, "\\/"))
	a.bindPathsInContainer()
	return a
}

// ConfigPath returns the path to the configuration directory.
func (a *Application) ConfigPath(path ...string) string {
	configDir := filepath.Join(a.basePath, "config")
	if len(path) > 0 && path[0] != "" {
		return filepath.Join(configDir, filepath.Clean(path[0]))
	}
	return configDir
}

// StoragePath returns the path to the storage directory.
func (a *Application) StoragePath(path ...string) string {
	storageDir := filepath.Join(a.basePath, "storage")
	if len(path) > 0 && path[0] != "" {
		return filepath.Join(storageDir, filepath.Clean(path[0]))
	}
	return storageDir
}

// PublicPath returns the path to the public directory.
func (a *Application) PublicPath(path ...string) string {
	publicDir := filepath.Join(a.basePath, "public")
	if len(path) > 0 && path[0] != "" {
		return filepath.Join(publicDir, filepath.Clean(path[0]))
	}
	return publicDir
}

// Register registers a service provider with the application.
func (a *Application) Register(provider support.ServiceProvider) {
	provider.Register(a.Container)
	a.providers = append(a.providers, provider)

	// If the application has already booted, we will boot this provider immediately.
	if a.booted {
		provider.Boot(a.Container)
	}
}

// Bootstrap boots the standard application bootstrappers.
func (a *Application) Bootstrap() {
	if a.bootstrapped {
		return
	}

	// 1. Load Environment Variables
	a.bootstrapEnvironmentVariables()

	// 2. Load and Mount Configuration Repository
	a.bootstrapConfiguration()

	a.bootstrapped = true
}

// BootstrapWith executes the given bootstrapper functions on the application.
func (a *Application) BootstrapWith(bootstrappers ...func(app *Application)) {
	for _, bootstrapper := range bootstrappers {
		bootstrapper(a)
	}
	a.bootstrapped = true
}

// Boot boots the application's service providers.
func (a *Application) Boot() {
	if a.booted {
		return
	}

	// 1. Ensure environment variables and configs are bootstrapped
	a.Bootstrap()

	// 2. Boot all registered service providers
	for _, provider := range a.providers {
		provider.Boot(a.Container)
	}

	// 3. Sync configured environment, debug and timezone from the repository after providers booted
	a.bootstrapConfiguration()

	a.booted = true
}

// Environment returns true if the current environment matches any of the given environments.
func (a *Application) Environment(envs ...string) bool {
	if len(envs) == 0 {
		return a.env != ""
	}
	for _, e := range envs {
		if a.env == e {
			return true
		}
	}
	return false
}

// IsProduction checks if the application environment is production.
func (a *Application) IsProduction() bool {
	return a.env == "production"
}

// IsLocal checks if the application environment is local.
func (a *Application) IsLocal() bool {
	return a.env == "local"
}

// IsTesting checks if the application environment is testing.
func (a *Application) IsTesting() bool {
	return a.env == "testing"
}

// SetEnvironment sets the application environment.
func (a *Application) SetEnvironment(env string) {
	a.env = env
}

// GetEnvironment returns the current environment name.
func (a *Application) GetEnvironment() string {
	return a.env
}

// IsDebug checks if the application is running in debug mode.
func (a *Application) IsDebug() bool {
	return a.debug
}

// SetDebug sets the application debug mode.
func (a *Application) SetDebug(debug bool) {
	a.debug = debug
}

// SetTimezone sets the application default timezone.
func (a *Application) SetTimezone(tz string) {
	a.timezone = tz
}

// GetTimezone returns the application default timezone.
func (a *Application) GetTimezone() string {
	if a.timezone == "" {
		return "UTC"
	}
	return a.timezone
}

// GetLocation returns the parsed time.Location for the application timezone.
func (a *Application) GetLocation() (*time.Location, error) {
	return time.LoadLocation(a.GetTimezone())
}

// GetContainer returns the concrete *container.Container for Go 1.27 method generics access.
func (a *Application) GetContainer() *container.Container {
	return a.Container
}
