package provider

import (
	"github.com/go-think/think/container"
	"github.com/go-think/think/contract"
	"github.com/go-think/think/event"
)

// EventServiceProvider registers the event dispatcher.
type EventServiceProvider struct{}

// Register registers the event dispatcher into the container.
func (p *EventServiceProvider) Register(app *container.Container) {
	dispatcher := event.NewDispatcher()
	app.Instance[contract.EventDispatcher](dispatcher)
	app.Alias[contract.EventDispatcher]("events")
}

// Boot boots the event service provider.
func (p *EventServiceProvider) Boot(app *container.Container) {}
