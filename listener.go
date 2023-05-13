package eventful

import (
	"sync"
	"sync/atomic"
)

// Listener is a subscription to a signal
type Listener[T any] interface {
	// C returns the channel that will receive the events while the subscription is active
	C() <-chan T
	// Close unsubscribes from the event preventing the channel from receiving more events.
	// After a successful call to Close, eventually the channel is closed. Note that the channel may still have messages to be read before it is closed.
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
	addedChan       chan struct{}
	shuttingDown    atomic.Bool
}

func newSignalSubscription[T any](requestShutdown func(sub *signalSubscription[T]), input <-chan T, bufferSize int) *signalSubscription[T] {
	sub := &signalSubscription[T]{
		input:           input,
		requestShutdown: requestShutdown,
		output:          make(chan T, bufferSize),
		closeChan:       make(chan struct{}),
		launchedChan:    make(chan struct{}),
		addedChan:       make(chan struct{}),
	}
	go sub.loop()
	return sub
}

// WaitLaunch blocks until the subscription internal loop has launched
func (sub *signalSubscription[T]) WaitLaunch() {
	<-sub.launchedChan
}

// WaitCreation notifies the subscription has been added to the listeners
func (sub *signalSubscription[T]) NotifyAdded() {
	sub.WaitLaunch()
	close(sub.addedChan)
}

// WaitCreation blocks until the subscription has been added to the listeners
func (sub *signalSubscription[T]) WaitAdded() {
	<-sub.addedChan
}

func (sub *signalSubscription[T]) launchComplete() {
	close(sub.launchedChan)
}
func (sub *signalSubscription[T]) loop() {
	defer close(sub.output)
	sub.launchComplete()
	for !sub.shuttingDown.Load() {
		select {
		case v, open := <-sub.input:
			if !open {
				return
			}
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
	sub.beginShutdown()
}

func (sub *signalSubscription[T]) beginShutdown() {
	sub.mutex.Lock()
	defer sub.mutex.Unlock()
	requestShutdown := sub.requestShutdown
	if requestShutdown == nil {
		return
	}
	sub.shuttingDown.Store(true)
	sub.requestShutdown = nil
	go requestShutdown(sub)
}

// completeShutdown completes the shutdown of the subscription
// meaning it should close the output channel and stop listening to the input channel
func (sub *signalSubscription[T]) completeShutdown() {
	close(sub.closeChan)
}

func (sub *signalSubscription[T]) C() <-chan T {
	return sub.output
}
