package eventful

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSignal(t *testing.T) {
	ev := NewSignal[int]()
	wg := sync.WaitGroup{}
	received := []int{}
	receivedMutex := sync.Mutex{}
	sub := ev.Listeners().Listen()
	defer sub.Close()
	sub2 := ev.Listeners().Listen()
	defer sub2.Close()
	wg.Add(2)
	read := func(l Listener[int]) {
		v := <-l.C()
		receivedMutex.Lock()
		received = append(received, v)
		receivedMutex.Unlock()
		wg.Done()
	}
	go read(sub)
	go read(sub2)
	ev.Emit(10)
	wg.Wait()
	require.Equal(t, []int{10, 10}, received)
}

func TestSignalStream(t *testing.T) {

	ev := NewSignal[int](WithSignalBufferSize(1))
	wg := sync.WaitGroup{}
	sub := ev.Listeners().Listen()
	defer sub.Close()
	max := 1000
	wg.Add(max)
	received := []int{}
	go func() {
		for i := range sub.C() {
			t.Logf("sub received signal, i=%d", i)
			received = append(received, i)
			wg.Done()
		}
	}()
	go func() {
		for i := 0; i < max; i++ {
			t.Logf("emiting signal, i=%d", i)
			ev.Emit(i)
		}
	}()
	wg.Wait()
	time.Sleep(time.Second)
	require.Len(t, received, max)
}

func TestSignalClose(t *testing.T) {
	ev := NewSignal[int]()
	ev.Emit(20)
	subA := ev.Listeners().Listen()
	t.Logf("subA added")
	defer subA.Close()
	subB := ev.Listeners().Listen()
	t.Logf("subB added")
	defer subB.Close()
	t.Logf("closing signal")
	ev.Close()
	t.Logf("closed signal")
	<-subA.C()
	<-subB.C()
	t.Logf("read both channels closed")
}
