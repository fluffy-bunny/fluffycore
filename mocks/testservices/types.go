package testservices

import "net/http"

type MockBodyResponseFunc func(r *http.Request) ([]byte, int)

type MockRequestValidator func(r *http.Request) int

// MockResponse represents a response for the mock server to serve
type MockResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	BodyFunc   MockBodyResponseFunc
}

// MockServerProcedure ties a mock response to a url and a method
type MockServerProcedure struct {
	URI              string
	HTTPMethod       string
	Response         MockResponse
	RequestValidator MockRequestValidator
}

// MockRecorder provides a way to record request information from every successful request.
type MockRecorder interface {
	Record(r *http.Request)
}
type AppError struct {
	Id            string `json:"id"`
	Message       string `json:"message"`               // Message to be display to the end user without debugging information
	DetailedError string `json:"detailed_error"`        // Internal error string to help the developer
	RequestId     string `json:"request_id,omitempty"`  // The RequestId that's also set in the header
	StatusCode    int    `json:"status_code,omitempty"` // The http status code
	Where         string `json:"-"`                     // The function where it happened in the form of Struct.Func
	params        map[string]interface{}
}
