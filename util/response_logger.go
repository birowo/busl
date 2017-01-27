package util

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
)

// NewResponseLogger creates a new logger for HTTP responses
func NewResponseLogger(r *http.Request, w http.ResponseWriter) *ResponseLogger {
	return &ResponseLogger{ResponseWriter: w, request: r, status: http.StatusOK}
}

// ResponseLogger is a logger for HTTP responses
type ResponseLogger struct {
	http.ResponseWriter
	request *http.Request
	status  int
}

// WriteHeader writes a new header to the response
func (l *ResponseLogger) WriteHeader(s int) {
	l.ResponseWriter.WriteHeader(s)
	l.status = s
}

// CloseNotify returns a chan notifying when the connection is being closed
func (l *ResponseLogger) CloseNotify() <-chan bool {
	return l.ResponseWriter.(http.CloseNotifier).CloseNotify()
}

// Flush flushes the response
func (l *ResponseLogger) Flush() {
	l.ResponseWriter.(http.Flusher).Flush()
}

func (l *ResponseLogger) requestID() (id string) {
	if id = l.request.Header.Get("Request-Id"); id == "" {
		id = l.request.Header.Get("X-Request-Id")
	}

	if id == "" {
		// In the event of a rare case where uuid
		// generation fails, it's probably more
		// desirable to continue as is with an empty
		// request_id than to bubble the error up the
		// stack.
		uuid, _ := NewUUID()
		id = string(uuid)
	}

	return id
}

// WriteLog logs the response
func (l *ResponseLogger) WriteLog() {
	maskedStatus := strconv.Itoa(l.status/100) + "xx"
	CountWithData(fmt.Sprintf("http.status.%s", maskedStatus), 1, "request_id=%s", l.requestID())
	log.Printf("method=%s path=\"%s\" host=\"%s\" fwd=\"%s\" status=%d user_agent=\"%s\" request_id=%s",
		l.request.Method, l.request.URL.Path, l.request.Host, l.request.Header.Get("X-Forwarded-For"), l.status, l.request.UserAgent(), l.requestID())
}
