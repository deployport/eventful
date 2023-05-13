package eventful

// Listeners allows to listen to a signal. Signal owners can safely expose Listeners to the outside world.
type Listeners[T any] interface {
	Listen(opts ...ListenerOpt) Listener[T]
}

type listeners[T any] struct {
	fanout     chan T
	input      chan T
	addChan    chan *signalSubscription[T]
	removeChan chan *signalSubscription[T]
}

func newListeners[T any](bufferSize int) *listeners[T] {
	listeners := &listeners[T]{
		fanout:     make(chan T, bufferSize),
		input:      make(chan T, bufferSize),
		addChan:    make(chan *signalSubscription[T], 1),
		removeChan: make(chan *signalSubscription[T], 1),
	}
	go listeners.loop()
	return listeners
}

func (subs *listeners[T]) loop() {
	defer close(subs.fanout)
	listenerCount := 0
	for {
		select {
		case sub := <-subs.addChan:
			sub.WaitLaunch()
			listenerCount++
			sub.NotifyAdded()
		case sub := <-subs.removeChan:
			listenerCount--
			sub.completeShutdown()
		case v, open := <-subs.input:
			if !open {
				return
			}
			for i := 0; i < listenerCount; i++ {
				subs.fanout <- v // repeat the same value for each listener
			}
		}
	}
}

func (subs *listeners[T]) fire(v T) {
	subs.input <- v
}

func (subs *listeners[T]) Listen(opts ...ListenerOpt) Listener[T] {
	o := newListenerOptions()
	for _, opt := range opts {
		opt.Apply(&o)
	}
	sub := newSignalSubscription(subs.requestShutdown, subs.fanout, o.bufferSize)
	subs.addChan <- sub
	sub.WaitAdded()
	return sub
}

func (subs *listeners[T]) requestShutdown(sub *signalSubscription[T]) {
	subs.removeChan <- sub
}

// close closes the listeners
func (subs *listeners[T]) close() {
	close(subs.input)
}
