// SPDX-FileCopyrightText: 2025 GSI Helmholtzzentrum für Schwerionenforschung GmbH
//
// SPDX-License-Identifier: MPL-2.0

package testutil

// ErrorWriter is a test writer that returns errors.
// It can be configured to fail immediately or after a certain number of writes.
type ErrorWriter struct {
	err       error
	failAfter int // Number of writes before failing (0 = always fail)
	writes    int
}

// NewErrorWriter creates an ErrorWriter that always fails with the given error.
func NewErrorWriter(err error) *ErrorWriter {
	return &ErrorWriter{
		failAfter: 0,
		err:       err,
	}
}

// NewErrorWriterAfter creates an ErrorWriter that fails after n successful writes.
func NewErrorWriterAfter(n int, err error) *ErrorWriter {
	return &ErrorWriter{
		failAfter: n,
		err:       err,
	}
}

// Write implements io.Writer and returns an error based on configuration.
func (e *ErrorWriter) Write(p []byte) (n int, err error) {
	if e.failAfter == 0 || e.writes >= e.failAfter {
		return 0, e.err
	}

	e.writes++

	return len(p), nil
}
