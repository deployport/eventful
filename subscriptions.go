package eventful

import (
	"context"
	"sync"
)

// Subscriptions allows subscriptions to an event so functions participant in the event propagation. Event owners can safely expose Subscriptions to the outside world.
type Subscriptions[T any] interface {
	Subscribe(m func(ctx context.Context, v T) error) Subscription[T]
}

type subscriptions[T any] struct {
	subsCounter int
	subs        map[subID]*subscription[T]
	subsMutex   sync.RWMutex
}

// Subscribe subscribes to the event. The channel will receive the event
func (subs *subscriptions[T]) Subscribe(m func(ctx context.Context, v T) error) Subscription[T] {
	subs.subsMutex.Lock()
	defer subs.subsMutex.Unlock()
	subs.subsCounter++
	id := subID(subs.subsCounter)
	sub := &subscription[T]{
		id:          id,
		unsubscribe: subs.unsubscribe,
		m:           m,
	}
	subs.subs[id] = sub
	return sub
}

func (subs *subscriptions[T]) fire(ctx context.Context, v T) error {
	subs.subsMutex.RLock()
	defer subs.subsMutex.RUnlock()
	for _, sub := range subs.subs {
		if err := sub.fire(ctx, v); err != nil {
			return err
		}
	}
	return nil
}

func (subs *subscriptions[T]) unsubscribe(id subID) {
	subs.subsMutex.Lock()
	defer subs.subsMutex.Unlock()
	delete(subs.subs, id)
}
