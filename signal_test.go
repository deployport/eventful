package eventful

import (
	"sync"
	"sync/atomic"
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
	doneWithFinish := atomic.Int32{}
	finishedFire := atomic.Bool{}
	go func() {
		v := <-sub.C()
		receivedMutex.Lock()
		received = append(received, v)
		receivedMutex.Unlock()
		if finishedFire.Load() {
			doneWithFinish.Add(1)
		}
		wg.Done()
	}()
	go func() {
		v := <-sub2.C()
		receivedMutex.Lock()
		received = append(received, v)
		receivedMutex.Unlock()
		if finishedFire.Load() {
			doneWithFinish.Add(1)
		}
		wg.Done()
	}()
	testDone := make(chan bool)
	go func() {
		wg.Wait()
		testDone <- true
	}()
	ev.Emit(10)
	finishedFire.Store(true)
	<-testDone
	require.Equal(t, int32(2), doneWithFinish.Load())
	require.Len(t, received, 2)
	require.Equal(t, 10, received[0])
	require.Equal(t, 10, received[1])
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
	// ev.Emit(20)
	// ev.Emit(30)
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
