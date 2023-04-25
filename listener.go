package eventful

import (
	"sync"
)

// Listener is a subscription to a signal
type Listener[T any] interface {
	// C returns the channel that will receive the events while the subscription is active
	C() <-chan T
	// Close unsubscribes to the event in its own goroutine
	Close()
}

type subID int

type signalSubscription[T any] struct {
	id          subID
	unsubscribe func(id subID)
	c           chan T
	mutex       sync.Mutex
}

func (sub *signalSubscription[T]) fire(v T) {
	go sub.internalFire(v)
}

func (sub *signalSubscription[T]) internalFire(v T) {
	sub.mutex.Lock()
	defer sub.mutex.Unlock()
	if sub.c == nil {
		return
	}
	sub.c <- v
}

func (sub *signalSubscription[T]) Close() {
	sub.mutex.Lock()
	defer sub.mutex.Unlock()
	sub.closeLocked()
}

func (sub *signalSubscription[T]) closeLocked() {
	unsub := sub.unsubscribe
	if unsub == nil {
		return
	}
	go sub.unsubscribe(sub.id)
	sub.unsubscribe = nil
	close(sub.c)
	sub.c = nil
}

func (sub *signalSubscription[T]) C() <-chan T {
	return sub.c
}
