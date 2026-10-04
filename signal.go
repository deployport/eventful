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

// Emit fires the signal to all listeners in an unordered fashion. Every listener that exists when
// the signal takes v receives it exactly once. Emit returns once the signal has taken v, or once
// v fits the emit buffer. Delivery blocks rather than drops: a listener whose channel buffer is
// full blocks later values until it reads or is closed, so a listener that stops reading
// must be closed. Emit panics after Close.
func (ev *Signal[T]) Emit(v T) {
	ev.subs.fire(v)
}

// Listeners returns the subscriptions for this signal
func (ev *Signal[T]) Listeners() Listeners[T] {
	return ev.subs
}

// Close closes the signal. Values already emitted are still delivered, then every listener's
// channel is closed. A listener that is never read nor closed keeps the signal's loop alive
// until it is.
func (ev *Signal[T]) Close() {
	ev.subs.close()
}
