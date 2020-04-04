package main

import (
	"net/http"
	"os"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type config struct {
	Interfaces []string `mapstructure:"interfaces"`
}

const (
	version = "v0.5.3"
)

func main() {
	var ()
	// CLI arguments parsing
	kingpin.Version(version)
	kingpin.Parse()

	// Start up the logger
	var logger log.Logger
	logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "version", "v0.5.1", "caller", log.DefaultCaller)

	// Read the data from the config file
	// currently the following options can be used in the configuration folder
	// interfaces: array - array holding the dvice names
	logger.Log("msg", "reading config file ...")
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("/etc/tc_exporter/")
	viper.AddConfigPath("$HOME/.config/tc_exporter/")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		} else {
		}
	}
	var cf config
	err := viper.Unmarshal(&cf)
	if err != nil {
		logger.Log("level", "ERROR", "msg", "failed to read config file", "error", err)
	}
	logger.Log("msg", "successfully read config file")

	collector, err := tcexporter.NewTcCollector(cf.Interfaces, logger)
	if err != nil {
		logger.Log("msg", "failed to create TC collector", "err", err)
	}

	prometheus.MustRegister(collector)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Start listening for HTTP connections.
	logger.Log("msg", "starting TC exporter", "port", ":9704")
	if err := http.ListenAndServe(":9704", mux); err != nil {
		logger.Log("msg", "cannot start TC exporter", "err", err)
	}
}
