package errors

import (
	"fmt"
)

// ErrServerAlreadyRunning is an error returned by the server
// when Start is called on a n already started server.
var ErrServerAlreadyRunning = fmt.Errorf("server: already running")
