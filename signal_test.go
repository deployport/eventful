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

func TestSignalListenOnce(t *testing.T) {
	ev := NewSignal[int](WithSignalBufferSize(3))
	wg := sync.WaitGroup{}
	sub := ev.Listeners().Listen()
	defer sub.Close()
	wg.Add(1)
	doneWithFinish := atomic.Int32{}
	received := []int{}
	go func() {
		defer wg.Done()
		for v := range sub.C() {
			received = append(received, v)
			doneWithFinish.Add(1)
			sub.Close()
		}
	}()
	testDone := make(chan bool)
	go func() {
		wg.Wait()
		testDone <- true
	}()
	ev.Emit(10)
	ev.Emit(11)
	ev.Emit(12)
	require.True(t, <-testDone)
	require.Equal(t, int32(1), doneWithFinish.Load())
	require.Len(t, received, 1)
}

func TestSignalStream(t *testing.T) {
	ev := NewSignal[int]()
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

func TestSignalStreamPort(t *testing.T) {
	max := 1000
	ev := NewSignal[int]()
	go func() {
		for i := 0; i < max; i++ {
			ev.Emit(i)
		}
	}()
	wg := sync.WaitGroup{}
	subA := ev.Listeners().Listen()
	defer subA.Close()
	subB := ev.Listeners().Listen()
	defer subB.Close()
	wg.Add(max)
	received := []int{}
	go func() {
		for i := range subA.C() {
			t.Logf("sub A received signal, i=%d", i)
			received = append(received, i)
			wg.Done()
		}
		for i := range subB.C() {
			t.Logf("sub B received signal and then sleeping, i=%d", i)
			time.Sleep(time.Hour)
		}
	}()
	wg.Wait()
	time.Sleep(time.Second)
	require.Len(t, received, max)
}
