//revive:disable:var-naming
package misc

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/go-pkgz/lgr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var pusher *push.Pusher

// L is logger
var L = lgr.New(lgr.Msec, lgr.Debug, lgr.CallerFile, lgr.CallerFunc)

var taskErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "lafin_news_errors",
}, []string{"error"})

// InitMetrics initializes the metrics
func InitMetrics(url, job string) {
	if url != "" && job != "" {
		registry := prometheus.NewRegistry()
		registry.MustRegister(taskErrors)
		pusher = push.New(url, job).Gatherer(registry)
	}
}

// PushMetrics push metrics
func PushMetrics() {
	if pusher != nil {
		if err := pusher.Push(); err != nil {
			L.Logf("ERROR could not push to Pushgateway, %v", err)
		}
		taskErrors.Reset()
	}
}

// FormatGUID return formated GUID
func FormatGUID(path string) (string, error) {
	var r *regexp.Regexp
	r = regexp.MustCompile(`^\w+#\d+$`)
	if r.MatchString(path) {
		return path, nil
	}
	r = regexp.MustCompile(`err.*?/(\d+)$`)
	if r.MatchString(path) {
		return fmt.Sprintf("err#%s", r.FindStringSubmatch(path)[1]), nil
	}
	r = regexp.MustCompile(`delfi.*?/(\d+)/.*?$`)
	if r.MatchString(path) {
		return fmt.Sprintf("delfi#%s", r.FindStringSubmatch(path)[1]), nil
	}
	r = regexp.MustCompile(`delfi.*?id=(\d+)$`)
	if r.MatchString(path) {
		return fmt.Sprintf("delfi#%s", r.FindStringSubmatch(path)[1]), nil
	}
	return "", errors.New("empty GUID")

}
