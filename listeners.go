package eventful

import "reflect"

// Listeners allows to listen to a signal. Signal owners can safely expose Listeners to the outside world.
type Listeners[T any] interface {
	Listen(opts ...ListenerOpt) Listener[T]
}

// listeners owns the signal's delivery loop. Each listener has its own channel, and the loop is
// the only goroutine that sends on or closes it.
//
// Delivery blocks: an emitted value is offered to every listener that exists when the loop takes
// it, and the loop does not take the next value until each of those listeners has received it or
// has been closed. A listener whose buffer is full therefore blocks later Emits until it
// reads or is closed. Nothing is dropped.
//
// A listener's closed channel is the authority on whether it is closed. Close closes it and then
// puts a token in wake without blocking, so Close never waits on the loop. A pending delivery
// stops at once for a closed listener. The loop's main select takes the token and sweeps: every
// closed listener leaves the active set and its channel is closed. That bookkeeping happens only
// in the main select, so while a delivery to another, slow listener is pending, a closed
// listener's channel closes when that delivery ends.
type listeners[T any] struct {
	input   chan T
	addChan chan *signalSubscription[T]
	// wake has a buffer of one token. One token is enough: the sweep it causes reads every
	// listener's closed channel, so it covers every Close that happened before it.
	wake   chan struct{}
	done   chan struct{} // closed when the loop has returned
	active map[*signalSubscription[T]]struct{}
}

func newListeners[T any](bufferSize int) *listeners[T] {
	listeners := &listeners[T]{
		input:   make(chan T, bufferSize),
		addChan: make(chan *signalSubscription[T]),
		wake:    make(chan struct{}, 1),
		done:    make(chan struct{}),
		active:  map[*signalSubscription[T]]struct{}{},
	}
	go listeners.loop()
	return listeners
}

func (subs *listeners[T]) loop() {
	defer close(subs.done)
	for {
		select {
		case sub := <-subs.addChan:
			subs.add(sub)
		case <-subs.wake:
			subs.sweep()
		case v, open := <-subs.input:
			if !open {
				for sub := range subs.active {
					subs.remove(sub)
				}
				return
			}
			subs.deliver(v)
		}
	}
}

func (subs *listeners[T]) add(sub *signalSubscription[T]) {
	subs.active[sub] = struct{}{}
}

// sweep removes every closed listener.
func (subs *listeners[T]) sweep() {
	for sub := range subs.active {
		if sub.isClosed() {
			subs.remove(sub)
		}
	}
}

func (subs *listeners[T]) remove(sub *signalSubscription[T]) {
	delete(subs.active, sub)
	sub.closeOutput()
}

// deliver hands v to every open listener exactly once. It returns when each has received it or
// has been closed. A listener added meanwhile does not receive v.
//
// When every listener has room, deliver allocates nothing. The slow path is in deliverSlow,
// because it takes v's address for reflect, and that moves v to the heap.
func (subs *listeners[T]) deliver(v T) {
	var slow []*signalSubscription[T]
	for sub := range subs.active {
		if sub.isClosed() {
			continue
		}
		select {
		case sub.output <- v:
		default:
			if slow == nil {
				slow = make([]*signalSubscription[T], 0, len(subs.active))
			}
			slow = append(slow, sub)
		}
	}
	if len(slow) > 0 {
		subs.deliverSlow(v, slow)
	}
}

// deliverSlow blocks until each slow listener has received v or has been closed. It
// offers every send at once, so whichever listener reads first receives first, and it accepts
// Listen meanwhile, so Listen never waits behind a slow listener.
func (subs *listeners[T]) deliverSlow(v T, slow []*signalSubscription[T]) {
	const (
		addCase   = 0
		firstPair = 1 // each slow listener has a send case, then its closed case
	)
	value := reflect.ValueOf(&v).Elem()
	// slow only shrinks, so cases never grows past this capacity.
	cases := make([]reflect.SelectCase, firstPair, firstPair+2*len(slow))
	cases[addCase] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(subs.addChan)}
	for len(slow) > 0 {
		cases = cases[:firstPair]
		for _, sub := range slow {
			cases = append(cases,
				reflect.SelectCase{Dir: reflect.SelectSend, Chan: reflect.ValueOf(sub.output), Send: value},
				reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(sub.closed)},
			)
		}
		chosen, recv, _ := reflect.Select(cases)
		if chosen == addCase {
			subs.add(recv.Interface().(*signalSubscription[T]))
			continue
		}
		// Received, or closed: either way this listener is done with v.
		i := (chosen - firstPair) / 2
		slow = append(slow[:i], slow[i+1:]...)
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
	sub := newSignalSubscription[T](subs.notifyClosed, o.bufferSize)
	select {
	case subs.addChan <- sub:
	case <-subs.done:
		// The signal is closed: hand back a listener whose channel is already closed.
		sub.closeUnregistered()
	}
	return sub
}

// notifyClosed wakes the loop to sweep closed listeners. It never blocks: when a token is
// already waiting, the sweep it causes covers this Close too.
func (subs *listeners[T]) notifyClosed() {
	select {
	case subs.wake <- struct{}{}:
	default:
	}
}

// close closes the listeners
func (subs *listeners[T]) close() {
	close(subs.input)
}
