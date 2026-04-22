package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"net/http/pprof"

	"github.com/alecthomas/kingpin/v2"
	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
	"github.com/spf13/viper"
)

// Config datasructure representing the configuration file
type Config struct {
	NetNS   map[string]NS
	Filters tcexporter.FilterHolder `mapstructure:"filters"`
}

// NS holds a type alias so we can use it in the config file
type NS struct {
	Interfaces []string `name:"interfaces" mapstructure:"interfaces"`
}

var (
	configFile      = kingpin.Flag("config-file", "location of the config path").Default("").String()
	logLevel        = kingpin.Flag("log-level", "slog based log level").Default("info").String()
	listenAddress   = kingpin.Flag("listen-address", "address to listen on").Default(":9704").String()
	disableDefaults = kingpin.Flag("collector.disable-defaults", "Set all collectors to disabled by default.").Default("false").Bool()
)

func run(logger *slog.Logger, cfg Config) error {
	// registering application information
	prometheus.MustRegister(NewVersionCollector("tc_exporter"))

	// fetch all the interfaces from the configured network namespaces
	// and store them in a map
	netns := make(map[string][]rtnetlink.LinkMessage)
	for ns, sp := range cfg.NetNS {
		interfaces, err := getInterfacesInNetNS(sp.Interfaces, ns)
		if err != nil {
			slog.Error("failed to get interfaces from ns", "err", err, "netns", ns)
		}
		netns[ns] = interfaces
	}

	// initialise the collector with the configured subcollectors
	collector, err := tcexporter.NewTcCollector(netns, cfg.Filters, logger)
	if err != nil {
		slog.Error("failed to create TC collector", "err", err.Error())
		return err
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

	landingConfig := web.LandingConfig{
		Name:        "TC (Traffic Control) exporter",
		Description: "Prometheus TC Exporter",
		Version:     Version,
		Links: []web.LandingLinks{
			{
				Address: "/metrics",
				Text:    "Metrics",
			},
		},
	}
	landingPage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	mux.Handle("/", landingPage)

	// Start listening for HTTP connections.
	slog.Info("starting TC exporter", "listen-address", *listenAddress)
	if err := http.ListenAndServe(*listenAddress, mux); err != nil {
		slog.Error("cannot start TC exporter", "err", err.Error())
	}
	return nil
}

func main() {
	// CLI arguments parsing
	kingpin.CommandLine.Name = "tc-exporter"
	kingpin.CommandLine.Help = "prometheus exporter for linux traffic control"
	kingpin.HelpFlag.Short('h')
	kingpin.Version(Version)
	kingpin.Parse()

	if *disableDefaults {
		tcexporter.DisableDefaultCollectors()
	}

	var slogLevel slog.Level
	switch *logLevel {
	case "info":
		slogLevel = slog.LevelInfo
	case "error":
		slogLevel = slog.LevelError
	case "warn":
		slogLevel = slog.LevelWarn
	case "debug":
		slogLevel = slog.LevelDebug
	default:
		slogLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel}))
	slog.SetDefault(logger)

	var config Config
	viper.SetConfigName("config")             // name of config file (without extension)
	viper.SetConfigType("toml")               // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/etc/tc-exporter/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.tc-exporter") // call multiple times to add many search paths
	viper.AddConfigPath(".")                  // optionally look for config in the working directory
	if *configFile != "" {
		viper.AddConfigPath(*configFile)
	}
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		slog.Error("error reading config file", "err", err, "path", "test")
		os.Exit(10)
	}
	err = viper.Unmarshal(&config)
	if err != nil { // Handle errors reading the config file
		slog.Error("error unmarshalling config file", "err", err, "path", "test")
		os.Exit(11)
	}
	fmt.Println(config)

	if err := run(logger, config); err != nil {
		slog.Error("failed to run tc exporter", "error", err)
		os.Exit(2)
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
