package eventful

import "sync"

// Listener is a subscription to a signal
type Listener[T any] interface {
	// C returns the channel that will receive the events while the subscription is active
	C() <-chan T
	// Close unsubscribes from the event preventing the channel from receiving more events.
	// After a successful call to Close, eventually the channel is closed. Note that the channel may still have messages to be read before it is closed.
	// Close never waits on a delivery: a value pending to this listener is abandoned.
	Close()
}

type subID int

type signalSubscription[T any] struct {
	mutex  sync.Mutex    // makes Close idempotent
	closed chan struct{} // closed once the listener is closed
	wake   func()        // tells the signal's loop that a listener closed
	output chan T        // the signal's loop is its only sender
}

func newSignalSubscription[T any](wake func(), bufferSize int) *signalSubscription[T] {
	return &signalSubscription[T]{
		wake:   wake,
		closed: make(chan struct{}),
		output: make(chan T, bufferSize),
	}
}

// Close marks the listener closed and wakes the signal's loop, which closes the channel. Close
// never blocks: not on its own delivery, and not on a delivery to another listener. Values
// already in the buffer can still be read.
func (sub *signalSubscription[T]) Close() {
	sub.mutex.Lock()
	if sub.isClosed() {
		sub.mutex.Unlock()
		return
	}
	close(sub.closed)
	sub.mutex.Unlock()
	sub.wake()
}

// isClosed reports whether Close has been called. It never blocks.
func (sub *signalSubscription[T]) isClosed() bool {
	select {
	case <-sub.closed:
		return true
	default:
		return false
	}
}

// closeOutput closes the listener's channel. Call it only from the goroutine that sends on it
// (the signal's loop), or for a listener that never reached the loop.
func (sub *signalSubscription[T]) closeOutput() {
	close(sub.output)
}

// closeUnregistered closes a listener that never reached the signal's loop. Listen calls it
// before it hands the listener out, so nothing else can touch the listener meanwhile and no
// lock is needed. A later Close finds closed closed and returns; the channel reads as closed.
func (sub *signalSubscription[T]) closeUnregistered() {
	close(sub.closed)
	sub.closeOutput()
}

func (sub *signalSubscription[T]) C() <-chan T {
	return sub.output
}
