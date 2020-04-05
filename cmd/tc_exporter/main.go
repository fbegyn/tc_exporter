package main

import (
	"fmt"
	"net/http"
	"os"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type Config struct {
	ListenAddres string ``
	NetNS        map[int]NS
}

type NS struct {
	Interfaces []string
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
	// Set config locations
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("/etc/tc_exporter/")
	viper.AddConfigPath("$HOME/.config/tc_exporter/")
	viper.AddConfigPath(".")
	// Set defaults
	viper.SetDefault("listen-address", ":9704")
	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		} else {
		}
	}

	fmt.Println(viper.AllKeys())

	var cf Config
	cf.ListenAddres = viper.GetString("listen-address")
	err := viper.Unmarshal(&cf)
	if err != nil {
		logger.Log("level", "ERROR", "msg", "failed to read config file", "error", err)
	}
	logger.Log("msg", "succesfully read config file")

	fmt.Println(cf)

	fmt.Println(cf.ListenAddres)
	collectors := make(map[int]*prometheus.Collector)

	for ns, devices := range cf.NetNS {
		collector, err := tcexporter.NewTcCollector(ns, devices.Interfaces, logger)
		if err != nil {
			logger.Log("msg", "failed to create TC collector", "err", err)
		}
		collectors[ns] = &collector
	}

	prometheus.MustRegister(*collectors[0])

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Start listening for HTTP connections.
	logger.Log("msg", "starting TC exporter", "listen-address", cf.ListenAddres)
	if err := http.ListenAndServe(cf.ListenAddres, mux); err != nil {
		logger.Log("msg", "cannot start TC exporter", "err", err)
	}
}
