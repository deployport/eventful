package eventful_test

import (
	"context"
	"fmt"

	"github.com/deployport/eventful"
)

func ExampleEvent() {
	ev := eventful.NewEvent[string]()

	sub := ev.Subscriptions().Subscribe(func(ctx context.Context, v string) error {
		fmt.Printf("hello %s", v) // hello world
		return nil
	})
	defer sub.Close()

	sub = ev.Subscriptions().Subscribe(func(ctx context.Context, v string) error {
		fmt.Printf("world %s", v) // world hello
		return nil
	})
	defer sub.Close()

	ctx := context.Background()
	_ = ev.Trigger(ctx, "world") // blocking, this will trigger each subscription synchronously
}
