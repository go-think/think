package env

import (
	"os"

	"github.com/joho/godotenv"
)

// Load loads the .env file from the given path.
// It does not override variables that already exist in the environment.
func Load(filenames ...string) error {
	return godotenv.Load(filenames...)
}

// Overload loads the .env file and overrides existing environment variables.
func Overload(filenames ...string) error {
	return godotenv.Overload(filenames...)
}

// LoadWithPriority loads environment files with cascade priority:
// 1. Explicit filename or ENV_FILE environment variable
// 2. .env.{APP_ENV} if APP_ENV is defined and the file exists
// 3. Default .env file
func LoadWithPriority(customFile ...string) error {
	var targetFile string
	if len(customFile) > 0 && customFile[0] != "" {
		targetFile = customFile[0]
	} else if envFile := os.Getenv("ENV_FILE"); envFile != "" {
		targetFile = envFile
	}

	// 1. If an explicit target file is specified and exists, load it
	if targetFile != "" {
		if _, err := os.Stat(targetFile); err == nil {
			return Overload(targetFile)
		}
	}

	// 2. Try cascade loading .env.{APP_ENV} then fallback to .env
	appEnv := os.Getenv("APP_ENV")
	if appEnv != "" {
		specificFile := ".env." + appEnv
		if _, err := os.Stat(specificFile); err == nil {
			_ = Load(".env")
			return Overload(specificFile)
		}
	}

	// 3. Fallback to default .env file if it exists
	if _, err := os.Stat(".env"); err == nil {
		return Load(".env")
	}

	return nil
}

// Get gets an environment variable, returning a fallback if it isn't set.
func Get(key string, defaultVal ...string) string {
	if val, ok := os.LookupEnv(key); ok {
		return val
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

// GetBool gets an environment variable as a boolean
func GetBool(key string, defaultVal ...bool) bool {
	val := Get(key)
	if val == "" {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return false
	}
	return val == "true" || val == "1" || val == "yes" || val == "on"
}

// GetInt gets an environment variable as an integer
func GetInt(key string, defaultVal ...int) int {
	val := Get(key)
	if val == "" {
		if len(defaultVal) > 0 {
			return defaultVal[0]
		}
		return 0
	}
	var res int
	for _, c := range val {
		if c >= '0' && c <= '9' {
			res = res*10 + int(c-'0')
		}
	}
	return res
}
