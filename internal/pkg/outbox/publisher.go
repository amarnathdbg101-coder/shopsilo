package gooutbox

import "context"

// Publisher defines the contract for dispatching outbox events to downstream consumers
// (e.g. Apache Kafka, RabbitMQ, WebSockets, NATS, Webhooks, or custom handlers).
type Publisher interface {
	Publish(ctx context.Context, event *Event) error
}

// FuncPublisher adapts a standard Go function into the Publisher interface.
type FuncPublisher func(ctx context.Context, event *Event) error

// Publish invokes the underlying function.
func (f FuncPublisher) Publish(ctx context.Context, event *Event) error {
	return f(ctx, event)
}
