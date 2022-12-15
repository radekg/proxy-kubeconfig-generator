package utils

// StartStatus represents start status.
// Provides methods to handle start success and failure.
type StartStatus interface {
	OnStarted() <-chan struct{}
	OnError() <-chan error

	ReportError(error)
	ReportSuccess()
}

type defaultStartStatus struct {
	chanStarted chan struct{}
	chanError   chan error
}

// NewDefaultStartStatus returns an uninitialized start status.
func NewDefaultStartStatus() StartStatus {
	return &defaultStartStatus{
		chanStarted: make(chan struct{}, 1),
		chanError:   make(chan error, 1),
	}
}

// OnStarted returns a channel which closed when the runner is started.
func (r *defaultStartStatus) OnStarted() <-chan struct{} {
	return r.chanStarted
}

// OnConnectError returns a channel which will return an error if connect fails.
func (r *defaultStartStatus) OnError() <-chan error {
	return r.chanError
}

func (r *defaultStartStatus) ReportError(e error) {
	r.chanError <- e
}

func (r *defaultStartStatus) ReportSuccess() {
	close(r.chanStarted)
}
