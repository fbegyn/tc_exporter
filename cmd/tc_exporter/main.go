package main

import (
	"log/slog"
	"net/http"
	"os"

	"net/http/pprof"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"

	kingpin "github.com/alecthomas/kingpin/v2"
)

// Config datasructure representing the configuration file
type Config struct {
	LogLevel     slog.Level
	ListenAddres string
	NetNS        map[string]NS
}

// NS holds a type alias so we can use it in the config file
type NS struct {
	Interfaces []string
}

func main() {
	// CLI arguments parsing
	kingpin.Version(Version)
	kingpin.Parse()

	// Start up the logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Read the data from the config file
	// currently the following options can be used in the configuration folder
	// interfaces: array - array holding the dvice names
	logger.Info("reading config file")
	// Set config locations
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("/etc/tc_exporter/")
	viper.AddConfigPath("$HOME/.config/tc_exporter/")
	viper.AddConfigPath(".")
	// Set defaults
	viper.SetDefault("listen-address", ":9704")
	viper.SetDefault("log-level", slog.LevelInfo)
	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			logger.Error("could not find the config file")
		} else {
			logger.Error("something went wrong while reading the config", "err", err)
		}
	}

	var cf Config
	cf.ListenAddres = viper.GetString("listen-address")
	cf.LogLevel = slog.Level(viper.GetInt("log-level"))
	err := viper.Unmarshal(&cf)
	if err != nil {
		logger.Error("failed to read config file", "error", err)
	}
	logger.Info("successfully read config file")

	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cf.LogLevel}))
	slog.SetDefault(logger)

	// registering application information
	prometheus.MustRegister(NewVersionCollector("tc_exporter"))

	// fetch all the interfaces from the configured network namespaces
	// and store them in a map
	netns := make(map[string][]rtnetlink.LinkMessage)
	for ns, sp := range cf.NetNS {
		interfaces, err := getInterfacesInNetNS(sp.Interfaces, ns)
		if err != nil {
			slog.Error("failed to get interfaces from ns", "err", err, "netns", ns)
		}
		netns[ns] = interfaces
	}

	enabledCollectors := map[string]bool{
		"cbq":           true,
		"choke":         true,
		"codel":         true,
		"fq":            true,
		"fq_codel":      true,
		"hfsc_qdisc":    true,
		"service_curve": true,
		"htb":           true,
		"pie":           true,
		"red":           true,
		"sfb":           true,
		"sfq":           true,
	}

	// initialise the collector with the configured subcollectors
	collector, err := tcexporter.NewTcCollector(netns, enabledCollectors, logger)
	if err != nil {
		slog.Error("failed to create TC collector", "err", err.Error())
	}
	prometheus.MustRegister(collector)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))

	// Start listening for HTTP connections.
	slog.Info("starting TC exporter", "listen-address", cf.ListenAddres)
	if err := http.ListenAndServe(cf.ListenAddres, mux); err != nil {
		slog.Error("cannot start TC exporter", "err", err.Error())
	}
}

func getInterfacesInNetNS(devices []string, ns string) ([]rtnetlink.LinkMessage, error) {
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
