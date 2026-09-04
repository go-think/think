package console

import (
	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
)

// Kernel handles CLI console commands.
type Kernel struct {
	app *container.Container
}

// NewKernel returns a new Console Kernel instance.
func NewKernel(app *container.Container) contract.ConsoleKernel {
	return &Kernel{
		app: app,
	}
}

// Bootstrap bootstraps the application for console commands.
func (k *Kernel) Bootstrap() {
	if app := k.app.Make[contract.Application](); app != nil {
		if b, ok := app.(interface {
			HasBeenBootstrapped() bool
			Bootstrap()
			Boot()
		}); ok {
			if !b.HasBeenBootstrapped() {
				b.Bootstrap()
			}
			b.Boot()
		}
	}
}

// Handle handles an incoming console command and returns the exit status code.
func (k *Kernel) Handle(args ...string) int {
	k.Bootstrap()
	return 0
}

// Terminate terminates the application and command.
func (k *Kernel) Terminate(status int) {}
