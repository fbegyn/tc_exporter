package main

import (
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
	ListenAddress string                  `mapstructure:"listen-address"`
	LogLevel      string                  `mapstructure:"log-level"`
	NetNS         map[string]NS           `mapstructure:"netns"`
	Filters       tcexporter.FilterHolder `mapstructure:"filters"`
	Collector     map[string]bool         `mapstructure:"collector"`
}

// NS holds a type alias so we can use it in the config file
type NS struct {
	Interfaces []string `name:"interfaces" mapstructure:"interfaces"`
}

// explicitFlags records non-collector flags the user passed on the CLI, so we
// know to ignore the equivalent config-file values for those keys.
var explicitFlags = map[string]bool{}

func markExplicit(name string) kingpin.Action {
	return func(*kingpin.ParseContext) error {
		explicitFlags[name] = true
		return nil
	}
}

var (
	configFile      = kingpin.Flag("config-file", "location of the config path").Default("").String()
	logLevel        = kingpin.Flag("log-level", "slog based log level").Default("info").Action(markExplicit("log-level")).String()
	listenAddress   = kingpin.Flag("listen-address", "address to listen on").Default(":9704").Action(markExplicit("listen-address")).String()
	disableDefaults = kingpin.Flag("collector.disable-defaults", "Set all collectors to disabled by default.").Default("false").Action(markExplicit("collector.disable-defaults")).Bool()
)

func run(logger *slog.Logger, cfg Config, listen string) error {
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
	slog.Info("starting TC exporter", "listen-address", listen)
	if err := http.ListenAndServe(listen, mux); err != nil {
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

	// Read config first so its values can feed into logger setup and collector
	// state. CLI flags (already parsed) win over config; we track explicitness
	// via explicitFlags / forcedCollectors.
	var config Config
	viper.SetConfigName("config")             // name of config file (without extension)
	viper.SetConfigType("toml")               // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/etc/tc-exporter/")  // path to look for the config file in
	viper.AddConfigPath("$HOME/.tc-exporter") // call multiple times to add many search paths
	viper.AddConfigPath(".")                  // optionally look for config in the working directory
	if *configFile != "" {
		viper.AddConfigPath(*configFile)
	}
	if err := viper.ReadInConfig(); err != nil {
		slog.Error("error reading config file", "err", err)
		os.Exit(10)
	}
	if err := viper.Unmarshal(&config); err != nil {
		slog.Error("error unmarshalling config file", "err", err)
		os.Exit(11)
	}

	// Resolve log level: CLI explicit > config file > default ("info").
	level := *logLevel
	if !explicitFlags["log-level"] && config.LogLevel != "" {
		level = config.LogLevel
	}
	var slogLevel slog.Level
	switch level {
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

	// Resolve collector configuration. disable-defaults is a reserved key in
	// the [collector] section; CLI flag wins over config.
	disable := *disableDefaults
	if !explicitFlags["collector.disable-defaults"] {
		if v, ok := config.Collector["disable-defaults"]; ok {
			disable = v
		}
	}
	delete(config.Collector, "disable-defaults")
	if disable {
		tcexporter.DisableDefaultCollectors()
	}
	tcexporter.ApplyCollectorConfig(config.Collector, logger)

	// Resolve listen address: CLI explicit > config file > default.
	listen := *listenAddress
	if !explicitFlags["listen-address"] && config.ListenAddress != "" {
		listen = config.ListenAddress
	}

	if err := run(logger, config, listen); err != nil {
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
