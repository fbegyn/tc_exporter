package main

import (
	"net/http"

	"github.com/fbegyn/tc_exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type Config struct {
	Interfaces []string `mapstructure:"interfaces"`
}

func main() {
	var (
		promPort = kingpin.Flag("promport", "Port on which the prometheus exporter runs").Default("9601").Short('P').String()
	)
	// CLI arguments parsing
	kingpin.Version("v0.4.3")
	kingpin.Parse()

	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("/etc/tc_exporter/")
	viper.AddConfigPath("$HOME/.config/tc_exporter/")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logrus.Infoln("no config file found, using defaults")
		} else {
			logrus.Fatalln("failed to read config file")
		}
	}

	// Configuring the logging
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)

	logrus.Infof("reading config file\n")

	var cf Config
	viper.Unmarshal(&cf)

	logrus.Infoln("prometheus exporter enabled")

	http.Handle("/metrics", newHandler(100, cf.Interfaces))

	logrus.Fatal(http.ListenAndServe(":"+*promPort, nil))
}

type handler struct {
	unfilteredHandler       http.Handler
	exporterMetricsRegistry *prometheus.Registry
	maxRequests             int
	interfaces              []string
}

func newHandler(maxRequests int, interfaces []string) *handler {
	h := &handler{
		exporterMetricsRegistry: prometheus.NewRegistry(),
		maxRequests:             maxRequests,
		interfaces:              interfaces,
	}
	if innerHandler, err := h.innerHandler(); err != nil {
		logrus.Fatalf("Couldn't create metrics handler: %s", err)
	} else {
		h.unfilteredHandler = innerHandler
	}
	return h
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, err := h.innerHandler()
	if err != nil {
		logrus.Fatalf("failed to create http handler: %v", err)
	}
	handler.ServeHTTP(w, r)
}

func (h *handler) innerHandler() (http.Handler, error) {
	tc, err := collector.NewTcCollector(h.interfaces)
	if err != nil {
		logrus.Errorf("failed to create collector: %v", err)
		return nil, err
	}
	r := prometheus.NewRegistry()
	r.MustRegister(version.NewCollector("tc_exporter"))
	if err := r.Register(tc); err != nil {
		logrus.Errorf("couldn't register tc collector: %s", err)
		return nil, err
	}

	handler := promhttp.HandlerFor(
		prometheus.Gatherers{h.exporterMetricsRegistry, r},
		promhttp.HandlerOpts{
			ErrorHandling:       promhttp.ContinueOnError,
			MaxRequestsInFlight: h.maxRequests,
			Registry:            h.exporterMetricsRegistry,
		},
	)

	return handler, nil
}
