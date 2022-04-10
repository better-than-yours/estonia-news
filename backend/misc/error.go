package misc

import (
	"flag"

	"github.com/prometheus/client_golang/prometheus"
)

// Fatal expose fatal error
func Fatal(name, desc string, err error) {
	if flag.Lookup("test.v") == nil {
		taskErrors.With(prometheus.Labels{"error": name}).Inc()
		PushMetrics()
		L.Logf("FATAL %s, %v", desc, err)
	}
}

// Error expose error
func Error(name, desc string, err error) {
	if flag.Lookup("test.v") == nil {
		taskErrors.With(prometheus.Labels{"error": name}).Inc()
		L.Logf("ERROR %s, %v", desc, err)
	}
}
