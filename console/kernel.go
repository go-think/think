package console

import (
	"fmt"
	"os"

	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
)

// Kernel handles CLI console commands.
type Kernel struct {
	app       *container.Container
	routeDump func() (string, bool)
}

// NewKernel returns a new Console Kernel instance. routeDump provides the
// route-table dump for the route:list command (may be nil).
func NewKernel(app *container.Container, routeDump func() (string, bool)) contract.ConsoleKernel {
	return &Kernel{
		app:       app,
		routeDump: routeDump,
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

// routeDumpFn is provided by the kernel package so commands can read the
// route table without importing flow directly.

// dumpRoutes returns the route-table dump through the bound accessor.
func (k *Kernel) dumpRoutes() (string, bool) {
	if k.routeDump != nil {
		return k.routeDump()
	}
	return "", false
}

// Handle handles an incoming console command and returns the exit status code.
func (k *Kernel) Handle(args ...string) int {
	k.Bootstrap()

	if len(args) == 0 {
		k.help()
		return 0
	}
	for i := range k.commands() {
		if k.commands()[i].Name == args[0] {
			return k.commands()[i].Run(args[1:])
		}
	}
	fmt.Fprintf(os.Stderr, "command [%s] is not defined.\n", args[0])
	k.help()
	return 1
}

// Terminate terminates the application and command.
func (k *Kernel) Terminate(status int) {}
