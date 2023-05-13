package eventful

// SignalOptions are the options for the signal
type SignalOptions struct {
	emitBufferSize int
}

func newSignalOptions() SignalOptions {
	return SignalOptions{
		emitBufferSize: DefaultSignalBufferSize(),
	}
}

// SignalOpt is an option for the signal
type SignalOpt interface {
	Apply(options *SignalOptions)
}

// SignalOptFunc implements SignalOpt
type SignalOptFunc func(options *SignalOptions)

// Apply implements SignalOpt
func (f SignalOptFunc) Apply(options *SignalOptions) {
	f(options)
}

// DefaultSignalBufferSize returns the default buffer size for the signal. 0(unbuffered) is the default value.
func DefaultSignalBufferSize() int {
	return 0
}

// WithSignalBufferSize sets the buffer size for the signal. The default value is DefaultSignalBufferSize().
func WithSignalBufferSize(bufferSize int) SignalOpt {
	return SignalOptFunc(func(options *SignalOptions) {
		options.emitBufferSize = bufferSize
	})
}
