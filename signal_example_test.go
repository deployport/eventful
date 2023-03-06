package eventful_test

import (
	"fmt"

	"github.com/deployport/eventful"
)

func ExampleSignal() {
	ev := eventful.NewSignal[string]()

	listener := ev.Listeners().Listen()
	defer listener.Close()

	ev.Emit("hello") // non-blocking, this will trigger each listener on its own goroutine

	greeting := <-listener.C()
	fmt.Printf("%s world", greeting) // hello world
}
