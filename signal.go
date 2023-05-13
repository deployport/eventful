package eventful

// Signal is an special type of event that implements fan-out pattern. Multiple subscribers can be registered to a signal and all of them will be fired when the signal is emitted.
type Signal[T any] struct {
	subs *listeners[T]
}

// NewSignal creates a new Signal. The returned instance should not be exposed to the outside world, instead, the Listeners can be exposed to allow third-party listeners.
func NewSignal[T any](opts ...SignalOpt) *Signal[T] {
	o := newSignalOptions()
	for _, opt := range opts {
		opt.Apply(&o)
	}
	ev := Signal[T]{
		subs: newListeners[T](o.emitBufferSize),
	}
	return &ev
}

// Emit fires the signal to all listeners in an unordered fashion.
func (ev *Signal[T]) Emit(v T) {
	ev.subs.fire(v)
}

// Listeners returns the subscriptions for this signal
func (ev *Signal[T]) Listeners() Listeners[T] {
	return ev.subs
}

// Close closes
func (ev *Signal[T]) Close() {
	ev.subs.close()
}
