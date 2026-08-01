package scraper

import "fmt"

// StatusError wraps a scrape failure that carries an HTTP status code so
// callers can distinguish retryable (5xx/network) from permanent (4xx)
// failures.
type StatusError struct {
	StatusCode int
	Err        error
}

// Error implements the error interface.
func (e *StatusError) Error() string {
	return fmt.Sprintf("scrape returned status %d: %v", e.StatusCode, e.Err)
}

// Unwrap exposes the wrapped error for errors.As/Is.
func (e *StatusError) Unwrap() error {
	return e.Err
}
