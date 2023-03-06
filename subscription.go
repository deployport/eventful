package eventful

import (
	"context"
	"sync"
)

// Subscription is a subscription to an event
type Subscription[T any] interface {
	// Close unsubscribes to the event in its own goroutine
	Close()
}

type subscription[T any] struct {
	id          subID
	unsubscribe func(id subID)
	mutex       sync.Mutex
	m           func(ctx context.Context, v T) error
}

func (sub *subscription[T]) fire(ctx context.Context, v T) error {
	sub.mutex.Lock()
	defer sub.mutex.Unlock()
	if sub.m == nil {
		return nil
	}
	return sub.m(ctx, v)
}

func (sub *subscription[T]) Close() {
	sub.mutex.Lock()
	defer sub.mutex.Unlock()
	sub.closeLocked()
}

func (sub *subscription[T]) closeLocked() {
	unsub := sub.unsubscribe
	if unsub == nil {
		return
	}
	go sub.unsubscribe(sub.id)
	sub.unsubscribe = nil
	sub.m = nil
}
