package reporting

import (
	"log"
	"net/http"
)

// Reporter provides a general error reporting abstraction
type Reporter interface {

	// Error reports an error.  If the error occurred within the context of an
	// http request, additional details can be reported if the http.Request object
	// is provided
	Error(err error, request *http.Request) error

	// CapturePanic recovers from a panic and reports the error appropriately.
	// If the error occurred within the context of an http request, additional
	// details can be reported if the http.Request object is provided
	CapturePanic(request *http.Request) error
}

var reporters []Reporter

// RegisterReporter registers a reporter
func RegisterReporter(more ...Reporter) {
	reporters = append(reporters, more...)
}

// Error reports an error.  If the error occurred within the context of an
// http request, additional details can be reported if the http.Request object
// is provided
func Error(err error, request *http.Request) {
	for _, rep := range reporters {
		if err := rep.Error(err, request); err != nil {
			log.Printf("Problem capturing error: %v", err)
		}
	}
}

// CapturePanic recovers from a panic and reports the error appropriately.
// If the error occurred within the context of an http request, additional
// details can be reported if the http.Request object is provided
func CapturePanic(request *http.Request) {
	for _, rep := range reporters {
		if err := rep.CapturePanic(request); err != nil {
			log.Printf("Problem capturing panic: %v", err)
		}
	}
}
