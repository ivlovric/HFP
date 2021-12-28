package main

import (
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var connectedClients = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "hfp_client_connects_in",
		Help: "No of inbound client connects",
	},
)

var connectionStatus = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "hfp_connection_status_out",
		Help: "Connection status OUT - 1 is connected, 0 is disconnected",
	},
)

var hepBytesInFile = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "hfp_hep_bytes_in_file",
		Help: "No of HEP bytes in file",
	},
)

var hepFileFlushesSuccess = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "hfp_hep_file_flushes_success",
		Help: "No of times HEP pakets from file have been successfully sent over network to backend HEP server",
	},
)

var hepFileFlushesError = prometheus.NewCounter(
	prometheus.CounterOpts{
		Name: "hfp_hep_file_flushes_error",
		Help: "No of times HEP pakets from file failed sending over network to backend HEP server",
	},
)

var clientLastMetricTimestamp = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "hfp_client_in_last_metric_timestamp",
		Help: "Inbound client's last metric arrival",
	},
)

func startMetrics(wg *sync.WaitGroup) {
	prometheus.MustRegister(connectedClients)
	prometheus.MustRegister(connectionStatus)
	prometheus.MustRegister(hepBytesInFile)
	prometheus.MustRegister(hepFileFlushesSuccess)
	prometheus.MustRegister(hepFileFlushesError)
	prometheus.MustRegister(clientLastMetricTimestamp)

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":"+*PrometheusPort, nil)
	wg.Done()

}
