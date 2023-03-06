// package eventful provides a simple and generic way to handle Events & Signals in Go
//
// Events are ideal for:
//   - having multiple generics-typed subscriptions to the same event
//   - trigger values in an un-ordered, sequential and blocking manner
//   - return errors from subscriptions to stop event propagation
//   - expect errors from the event propagation
//   - cancel, timeout and deadline subscription execution using contexts
//   - only the owner can trigger events
//   - only subscriptions can close themselves(by only exposing Subscriptions to the outside world, package or enclosing type)
//
// Signals are ideal for:
//   - having multiple generics-typed listeners, with fan-out semantics
//   - emit values in an un-ordered, non-sequential and non-blocking manner
//   - uses unbuffered channels
//   - only the owner can emit signals
//   - only listeners can close themselves (by only exposing Listeners to the outside world, package or enclosing type)
package eventful
