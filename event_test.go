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
	doneWithFinish := atomic.Int32{}
	sub := ev.Subscriptions().Subscribe(func(ctx context.Context, v int) error {
		doneWithFinish.Add(1)
		return fmt.Errorf("error returned")
	})
	defer sub.Close()
	sub2 := ev.Subscriptions().Subscribe(func(ctx context.Context, v int) error {
		doneWithFinish.Add(1)
		return nil
	})
	defer sub2.Close()

	testDone := make(chan error)
	go func() {
		testDone <- ev.Trigger(ctx, 10)
	}()
	err := <-testDone
	require.NotNil(t, err, "expected error to be returned from a subscription")
	require.Equal(t, int32(1), doneWithFinish.Load())
}
