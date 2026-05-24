package entity

import "errors"

// Coded is implemented by errors that carry a machine-readable kernel
// error code — the verb-error analogue of check.Finding's Code field.
// It is a behavioral interface (cf. net.Error's Timeout): any typed
// error can advertise its code without sharing a base type. Consumers
// extract the code with [Code], which walks the error chain.
type Coded interface {
	error
	Code() string
}

// Code returns the kernel error code carried by err, if any error in
// its Unwrap chain implements [Coded]. The bool reports whether a code
// was found, which distinguishes "no Coded error in the chain" from a
// Coded error whose code is the empty string.
func Code(err error) (string, bool) {
	var c Coded
	if errors.As(err, &c) {
		return c.Code(), true
	}
	return "", false
}
