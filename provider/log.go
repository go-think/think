package provider

import (
	"github.com/go-think/log"
	"github.com/go-think/log/record"
	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
)

// LogServiceProvider registers the logger.
type LogServiceProvider struct{}

// Register registers the logger into the container using lazy resolution.
func (p *LogServiceProvider) Register(app *container.Container) {
	loggerFactory := func() contract.Logger {
		channel := "develop"
		level := record.DEBUG

		cfg := app.Make[contract.Config]()
		if cfg != nil {
			if ch := cfg.GetString("logging.default"); ch != "" {
				channel = ch
			}
		}

		return log.NewLogger(channel, level)
	}

	app.Singleton[contract.Logger](loggerFactory)
	app.Alias("logger", "Logger")
	app.Alias("log", "Logger")
}

// Boot boots the log service provider.
func (p *LogServiceProvider) Boot(app *container.Container) {}
