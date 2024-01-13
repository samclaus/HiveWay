package main

import "fmt"

// ErrorWithCode is a standard error for giving feedback to clients.
type ErrorWithCode struct {
	Code    string      `msgpack:"code"`
	Message string      `msgpack:"message"`
	Details interface{} `msgpack:"details"`
}

// Error implements the primitive error interface.
func (err *ErrorWithCode) Error() string {
	return fmt.Sprintf("[%s] %s", err.Code, err.Message)
}
