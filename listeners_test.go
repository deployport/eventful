package eventful_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/deployport/eventful"
)

const deadline = 5 * time.Second

func within(t *testing.T, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(deadline):
		t.Fatalf("hang: %s did not finish within %s", what, deadline)
	}
}

func listen[T any](t *testing.T, sig *eventful.Signal[T], opts ...eventful.ListenerOpt) eventful.Listener[T] {
	t.Helper()
	var l eventful.Listener[T]
	within(t, "Listen", func() { l = sig.Listeners().Listen(opts...) })
	return l
}

func closeListener[T any](t *testing.T, l eventful.Listener[T]) {
	t.Helper()
	within(t, "listener Close", l.Close)
}

func collect[T any](l eventful.Listener[T], slow time.Duration) <-chan []T {
	out := make(chan []T, 1)
	go func() {
		var got []T
		for v := range l.C() {
			got = append(got, v)
			if slow > 0 {
				time.Sleep(slow)
			}
		}
		out <- got
	}()
	return out
}

func wantSequence(t *testing.T, name string, got []int, n int) {
	t.Helper()
	if len(got) != n {
		t.Fatalf("%s received %d events, want %d exactly once each: %v", name, len(got), n, got)
	}
	for i, v := range got {
		if v != i {
			t.Fatalf("%s received %v, want 0..%d in order, each exactly once", name, got, n-1)
		}
	}
}

func TestListenerClosedDuringEmitNeverBlocksEmit(t *testing.T) {
	sig := eventful.NewSignal[int]()
	l := listen(t, sig)
	emitted := make(chan struct{})
	go func() {
		defer close(emitted)
		for i := 0; i < 3; i++ {
			sig.Emit(i)
		}
	}()
	time.Sleep(50 * time.Millisecond)
	closeListener(t, l)
	select {
	case <-emitted:
	case <-time.After(deadline):
		t.Fatalf("hang: Emit still blocked %s after its only listener closed", deadline)
	}
	within(t, "the closed listener's channel closing", func() {
		for range l.C() {
		}
	})
	sig.Close()
}

func TestListenCloseChurnUnderContinuousEmit(t *testing.T) {
	for _, buffer := range []int{0, 1, 4} {
		t.Run(fmt.Sprintf("buffer %d", buffer), func(t *testing.T) {
			sig := eventful.NewSignal[int]()
			stop := make(chan struct{})
			emitterDone := make(chan struct{})
			go func() {
				defer close(emitterDone)
				for i := 0; ; i++ {
					select {
					case <-stop:
						return
					default:
					}
					sig.Emit(i)
				}
			}()
			within(t, "500 listen/read-one/close rounds", func() {
				for i := 0; i < 500; i++ {
					l := sig.Listeners().Listen(eventful.WithListenerBufferSize(buffer))
					<-l.C()
					l.Close()
				}
			})
			close(stop)
			select {
			case <-emitterDone:
			case <-time.After(deadline):
				t.Fatalf("hang: Emit blocked with no listener left")
			}
			sig.Close()
		})
	}
}

func TestListenerWithFullBufferClosedDuringEmit(t *testing.T) {
	sig := eventful.NewSignal[int]()
	l := listen(t, sig, eventful.WithListenerBufferSize(1))
	emitted := make(chan struct{})
	go func() {
		defer close(emitted)
		for i := 0; i < 3; i++ {
			sig.Emit(i)
		}
	}()
	time.Sleep(50 * time.Millisecond)
	closeListener(t, l)
	select {
	case <-emitted:
	case <-time.After(deadline):
		t.Fatalf("hang: Emit still blocked after the full listener closed")
	}
	select {
	case got := <-collect(l, 0):
		if len(got) != 1 || got[0] != 0 {
			t.Fatalf("closed listener read %v, want [0], the value its buffer held", got)
		}
	case <-time.After(deadline):
		t.Fatalf("hang: the closed listener's channel never closed")
	}
	sig.Close()
}

func TestCloseNeverWaitsBehindAnotherSlowListener(t *testing.T) {
	sig := eventful.NewSignal[int]()
	slow := listen(t, sig)
	var fast []eventful.Listener[int]
	var results []<-chan []int
	for i := 0; i < 3; i++ {
		l := listen(t, sig)
		fast = append(fast, l)
		results = append(results, collect(l, 0))
	}
	emitted := make(chan struct{})
	go func() {
		defer close(emitted)
		sig.Emit(0)
		sig.Emit(1)
	}()
	time.Sleep(50 * time.Millisecond)
	for _, l := range fast {
		closeListener(t, l)
	}
	closeListener(t, slow)
	select {
	case <-emitted:
	case <-time.After(deadline):
		t.Fatalf("hang: Emit still blocked after every listener closed")
	}
	for i, res := range results {
		select {
		case got := <-res:
			if len(got) != 1 || got[0] != 0 {
				t.Fatalf("fast listener %d read %v, want [0]", i, got)
			}
		case <-time.After(deadline):
			t.Fatalf("hang: fast listener %d channel never closed", i)
		}
	}
	sig.Close()
}

func TestClosedListenerChannelClosesWithoutFurtherEvents(t *testing.T) {
	sig := eventful.NewSignal[int]()
	defer sig.Close()
	l := listen(t, sig)
	within(t, "one Emit", func() { sig.Emit(7) })
	select {
	case got := <-l.C():
		if got != 7 {
			t.Fatalf("listener read %d, want 7", got)
		}
	case <-time.After(deadline):
		t.Fatalf("hang: the listener never received the one event")
	}
	closeListener(t, l)
	within(t, "ranging over the closed listener's channel with no further event", func() {
		for range l.C() {
		}
	})
}

func TestEveryListenerReceivesEveryEventExactlyOnce(t *testing.T) {
	const listeners = 6
	const events = 100
	sig := eventful.NewSignal[int]()
	results := make([]<-chan []int, listeners)
	for i := range results {
		slow := time.Duration(0)
		if i == 0 {
			slow = time.Millisecond
		}
		results[i] = collect(listen(t, sig), slow)
	}
	within(t, "emitting to listeners that all read", func() {
		for i := 0; i < events; i++ {
			sig.Emit(i)
		}
		sig.Close()
	})
	for i, res := range results {
		select {
		case got := <-res:
			wantSequence(t, fmt.Sprintf("listener %d", i), got, events)
		case <-time.After(deadline):
			t.Fatalf("hang: listener %d channel never closed after the signal closed", i)
		}
	}
}

func TestClosedSilentListenerDoesNotBlockOthers(t *testing.T) {
	const events = 50
	sig := eventful.NewSignal[int]()
	silent := listen(t, sig)
	var mu sync.Mutex
	firstSeen := sync.WaitGroup{}
	firstSeen.Add(2)
	readers := make([]<-chan []int, 2)
	for i := range readers {
		l := listen(t, sig)
		out := make(chan []int, 1)
		readers[i] = out
		go func() {
			var got []int
			for v := range l.C() {
				mu.Lock()
				got = append(got, v)
				if len(got) == 1 {
					firstSeen.Done()
				}
				mu.Unlock()
			}
			out <- got
		}()
	}
	emitted := make(chan struct{})
	go func() {
		defer close(emitted)
		for i := 0; i < events; i++ {
			sig.Emit(i)
		}
	}()
	within(t, "both readers receiving the first event", firstSeen.Wait)
	closeListener(t, silent)
	select {
	case <-emitted:
	case <-time.After(deadline):
		t.Fatalf("hang: Emit still blocked after the silent listener closed")
	}
	sig.Close()
	for i, res := range readers {
		select {
		case got := <-res:
			wantSequence(t, fmt.Sprintf("reader %d", i), got, events)
		case <-time.After(deadline):
			t.Fatalf("hang: reader %d channel never closed", i)
		}
	}
}

func TestListenAndCloseAfterSignalCloseReturn(t *testing.T) {
	sig := eventful.NewSignal[int]()
	before := listen(t, sig)
	sig.Close()
	within(t, "a live listener's channel closing with the signal", func() {
		for range before.C() {
		}
	})
	closeListener(t, before)
	after := listen(t, sig)
	within(t, "the channel of a listener opened after the signal closed closing", func() {
		for range after.C() {
		}
	})
	closeListener(t, after)
}

func TestDeliveryToListenersWithRoomAllocatesNothing(t *testing.T) {
	sig := eventful.NewSignal[int]()
	defer sig.Close()
	var ls []eventful.Listener[int]
	for i := 0; i < 8; i++ {
		l := listen(t, sig, eventful.WithListenerBufferSize(1))
		defer l.Close()
		ls = append(ls, l)
	}
	emitAndDrain := func() {
		sig.Emit(1)
		for _, l := range ls {
			<-l.C()
		}
	}
	within(t, "one event reaching every listener", emitAndDrain)
	allocs := testing.AllocsPerRun(1000, emitAndDrain)
	if allocs != 0 {
		t.Fatalf("delivery to listeners with room allocated %v times per event, want 0", allocs)
	}
}
