package host

import "errors"

var errHostHTTP = errors.New("host http_request failed")

type hostError struct{ msg string }

func (e *hostError) Error() string { return e.msg }
