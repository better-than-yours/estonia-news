package misc

import (
	"github.com/go-pkgz/lgr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var pusher *push.Pusher

// L is logger
var L = lgr.New(lgr.Msec, lgr.Debug, lgr.CallerFile, lgr.CallerFunc)

// TaskErrors is error metrics
var TaskErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "lafin_news_errors",
}, []string{"error"})

// InitMetrics initializes the metrics
func InitMetrics(url, job string) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(TaskErrors)
	pusher = push.New(url, job).Gatherer(registry)
}

// PushMetrics push metrics
func PushMetrics() {
	if err := pusher.Push(); err != nil {
		L.Logf("ERROR could not push to Pushgateway, %v", err)
	}
	TaskErrors.Reset()
}
