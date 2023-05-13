package eventful

import (
	"sync"
	"sync/atomic"
)

// Listener is a subscription to a signal
type Listener[T any] interface {
	// C returns the channel that will receive the events while the subscription is active
	C() <-chan T
	// Close unsubscribes from the event preventing the channel from receiving more events
	Close()
}

type subID int

type signalSubscription[T any] struct {
	requestShutdown func(sub *signalSubscription[T])
	output          chan T
	input           <-chan T
	mutex           sync.Mutex
	closeChan       chan struct{}
	launchedChan    chan struct{}
	shuttingDown    atomic.Bool
}

func newSignalSubscription[T any](requestShutdown func(sub *signalSubscription[T]), input <-chan T) *signalSubscription[T] {
	sub := &signalSubscription[T]{
		input:           input,
		requestShutdown: requestShutdown,
		output:          make(chan T), // TODO: use buffer
		closeChan:       make(chan struct{}),
		launchedChan:    make(chan struct{}),
	}
	go sub.loop()
	return sub
}

// WaitLaunch blocks until the subscription internal loop has launched
func (sub *signalSubscription[T]) WaitLaunch() {
	<-sub.launchedChan
}

func (sub *signalSubscription[T]) launchComplete() {
	close(sub.launchedChan)
}
func (sub *signalSubscription[T]) loop() {
	defer close(sub.output)
	sub.launchComplete()
	for {
		select {
		case v := <-sub.input:
			if sub.shuttingDown.Load() {
				continue
			}
			sub.output <- v
		case <-sub.closeChan:
			return
		}
	}
}

func (sub *signalSubscription[T]) Close() {
	sub.mutex.Lock()
	defer sub.mutex.Unlock()
	sub.beginShutdown()
}

func (sub *signalSubscription[T]) beginShutdown() {
	requestShutdown := sub.requestShutdown
	if requestShutdown == nil {
		return
	}
	sub.shuttingDown.Store(true)
	requestShutdown(sub)
	sub.requestShutdown = nil
}

// completeShutdown completes the shutdown of the subscription
// meaning it should close the output channel and stop listening to the input channel
func (sub *signalSubscription[T]) completeShutdown() {
	close(sub.closeChan)
}

func (sub *signalSubscription[T]) C() <-chan T {
	return sub.output
}
