package eventful

// ListenerOptions are the options for the signal listener
type ListenerOptions struct {
	bufferSize int
}

func newListenerOptions() ListenerOptions {
	return ListenerOptions{
		bufferSize: DefaultSignalBufferSize(),
	}
}

// ListenerOpt is an option for the signal
type ListenerOpt interface {
	Apply(options *ListenerOptions)
}

// ListenerOptFunc implements ListenerOpt
type ListenerOptFunc func(options *ListenerOptions)

// Apply implements ListenerOpt
func (f ListenerOptFunc) Apply(options *ListenerOptions) {
	f(options)
}

// WithListenerBufferSize sets the buffer size for the signal listener. The default value is DefaultSignalBufferSize().
// While the buffer is full, the signal delays further values for every listener until this
// one reads or is closed.
func WithListenerBufferSize(bufferSize int) ListenerOpt {
	return ListenerOptFunc(func(options *ListenerOptions) {
		options.bufferSize = bufferSize
	})
}
