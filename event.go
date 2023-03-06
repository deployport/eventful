package eventful

import "context"

// Event is a type that can be fired and subscribed to. All subscriptions are fired synchronously.
type Event[T any] struct {
	subs subscriptions[T]
}

// NewEvent creates a new Event. The returned instance should not be exposed to the outside world, instead, the Subscriptions can be exposed to allow third-party subscriptions.
func NewEvent[T any]() *Event[T] {
	ev := Event[T]{
		subs: subscriptions[T]{
			subs: map[subID]*subscription[T]{},
		},
	}
	return &ev
}

// Trigger fires the event and all the subscribers will be called synchronously.
func (ev *Event[T]) Trigger(ctx context.Context, v T) error {
	return ev.subs.fire(ctx, v)
}

// Subscriptions returns the subscriptions for this event
func (ev *Event[T]) Subscriptions() Subscriptions[T] {
	return &ev.subs
}
