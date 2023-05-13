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
func WithListenerBufferSize(bufferSize int) ListenerOpt {
	return ListenerOptFunc(func(options *ListenerOptions) {
		options.bufferSize = bufferSize
	})
}
