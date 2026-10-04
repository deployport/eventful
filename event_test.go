package eventful

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEvent(t *testing.T) {
	ctx := context.Background()
	ev := NewEvent[int]()
	wg := sync.WaitGroup{}
	received := []int{}
	wg.Add(2)
	doneWithFinish := atomic.Int32{}
	finishedFire := atomic.Bool{}
	sub := ev.Subscriptions().Subscribe(func(ctx context.Context, v int) error {
		wg.Done()
		received = append(received, v)
		if finishedFire.Load() {
			doneWithFinish.Add(1)
		}
		return nil
	})
	defer sub.Close()
	sub2 := ev.Subscriptions().Subscribe(func(ctx context.Context, v int) error {
		wg.Done()
		received = append(received, v)
		if finishedFire.Load() {
			doneWithFinish.Add(1)
		}
		return nil
	})
	defer sub2.Close()

	testDone := make(chan bool)
	go func() {
		wg.Wait()
		testDone <- true
	}()
	err := ev.Trigger(ctx, 10)
	require.True(t, <-testDone)
	require.Nil(t, err, "should have returned no errors")
	finishedFire.Store(true)
	require.Equal(t, int32(0), doneWithFinish.Load())
}

func TestEventError(t *testing.T) {
	ctx := context.Background()
	ev := NewEvent[int]()
	ran := atomic.Int32{}
	fail := func(ctx context.Context, v int) error {
		ran.Add(1)
		return fmt.Errorf("error returned")
	}
	sub := ev.Subscriptions().Subscribe(fail)
	defer sub.Close()
	sub2 := ev.Subscriptions().Subscribe(fail)
	defer sub2.Close()

	err := ev.Trigger(ctx, 10)
	require.NotNil(t, err, "expected error to be returned from a subscription")
	require.Equal(t, int32(1), ran.Load(), "a subscription ran after another returned an error")
}
