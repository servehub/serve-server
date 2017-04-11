package webhooks

import (
	"fmt"
	"net/http"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/pressly/chi/middleware"
)

type StructuredLogger struct {
	log *logrus.Entry
}

func (l *StructuredLogger) NewLogEntry(r *http.Request) middleware.LogEntry {
	return &StructuredLoggerEntry{
		r.Method + " " + r.RequestURI,
		l.log.WithFields(logrus.Fields{
			"remote_addr":     r.RemoteAddr,
			"request_method":  r.Method,
			"uri":             r.RequestURI,
			"hostname":        r.Host,
			"http_user_agent": r.UserAgent(),
		}),
	}
}

type StructuredLoggerEntry struct {
	request string
	log     *logrus.Entry
}

func (l *StructuredLoggerEntry) Write(status, bytes int, elapsed time.Duration) {
	l.log.WithFields(logrus.Fields{
		"status":          status,
		"body_bytes_sent": bytes,
		"request_time":    float64(elapsed.Nanoseconds()) / 1000000.0,
	}).Infof("%s --> %d %s, %s", l.request, status, http.StatusText(status), elapsed)
}

func (l *StructuredLoggerEntry) Panic(v interface{}, stack []byte) {
	l.log = l.log.WithFields(logrus.Fields{
		"stack": string(stack),
		"panic": fmt.Sprintf("%+v", v),
	})
}
