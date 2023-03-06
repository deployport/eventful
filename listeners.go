package eventful

import (
	"sync"
)

// Listeners allows to listen to a signal. Signal owners can safely expose Listeners to the outside world.
type Listeners[T any] interface {
	Listen() Listener[T]
}

type listeners[T any] struct {
	subsCounter int
	subs        map[subID]*signalSubscription[T]
	subsMutex   sync.RWMutex
}

func (subs *listeners[T]) fire(v T) {
	subs.subsMutex.RLock()
	defer subs.subsMutex.RUnlock()
	for _, sub := range subs.subs {
		sub.fire(v)
	}
}

func (subs *listeners[T]) Listen() Listener[T] {
	subs.subsMutex.Lock()
	defer subs.subsMutex.Unlock()
	subs.subsCounter++
	id := subID(subs.subsCounter)
	sub := &signalSubscription[T]{
		id:          id,
		unsubscribe: subs.unsubscribe,
		c:           make(chan T),
	}
	subs.subs[id] = sub
	return sub
}

func (subs *listeners[T]) unsubscribe(id subID) {
	subs.subsMutex.Lock()
	defer subs.subsMutex.Unlock()
	delete(subs.subs, id)
}
