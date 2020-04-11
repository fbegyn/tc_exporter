package main

import (
	"fmt"
	"net/http"
	"os"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/go-kit/kit/log"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type Config struct {
	ListenAddres string ``
	NetNS        map[string]NS
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
	logger = log.With(logger, "ts", log.DefaultTimestampUTC, "version", "v0.6.0-rc0", "caller", log.DefaultCaller)

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

	var cf Config
	cf.ListenAddres = viper.GetString("listen-address")
	err := viper.Unmarshal(&cf)
	if err != nil {
		logger.Log("level", "ERROR", "msg", "failed to read config file", "error", err)
	}
	logger.Log("msg", "successfully read config file")

	netns := make(map[string][]rtnetlink.LinkMessage)
	for ns, sp := range cf.NetNS {
		interfaces, err := getInterfaceInNS(sp.Interfaces, ns)
		if err != nil {
			logger.Log("msg", "failed to get interfaces from ns", "err", err, "netns", ns)
		}
		netns[ns] = interfaces
	}

	fmt.Println(netns)

	collector, err := tcexporter.NewTcCollector(netns, logger)
	if err != nil {
		logger.Log("msg", "failed to create TC collector", "err", err)
	}

	prometheus.MustRegister(collector)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	// Start listening for HTTP connections.
	logger.Log("msg", "starting TC exporter", "listen-address", cf.ListenAddres)
	if err := http.ListenAndServe(cf.ListenAddres, mux); err != nil {
		logger.Log("msg", "cannot start TC exporter", "err", err)
	}
}

func getInterfaceInNS(devices []string, ns string) ([]rtnetlink.LinkMessage, error) {
	con, err := tcexporter.GetNetlinkConn(ns)
	if err != nil {
		return nil, err
	}
	defer con.Close()

	links, err := con.Link.List()
	if err != nil {
		return nil, err
	}

	selected := make([]rtnetlink.LinkMessage, len(devices))
	for _, link := range links {
		for i, interf := range devices {
			if interf == link.Attributes.Name {
				selected[i] = link
			}
		}
	}

	return selected, nil
}
