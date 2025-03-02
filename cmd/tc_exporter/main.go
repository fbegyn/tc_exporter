package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"net/http/pprof"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/alecthomas/kong"
	kongtoml "github.com/alecthomas/kong-toml"
)

// Config datasructure representing the configuration file
type Config struct {
	LogLevel     slog.Level
	ListenAddres string
	NetNS        map[string]NS
}

var cli App

type App struct {
	Config       string          `help:"location of the config path" default:"config.toml" name:"config-file"`
	LogLevel     string          `help:"slog based log level" default:"info" name:"log-level"`
	ListenAddres string          `help:"address to listen on" default:":9704" name:"listen-address"`
	Collector    map[string]bool `name:"collector" help:"collectors to enable"`
	NetNS        map[string]NS   `name:"netns"`
}

// NS holds a type alias so we can use it in the config file
type NS struct {
	Interfaces []string `name:"interfaces"`
}

func (a *App) Run(logger *slog.Logger) error {
	// registering application information
	prometheus.MustRegister(NewVersionCollector("tc_exporter"))

	// fetch all the interfaces from the configured network namespaces
	// and store them in a map
	netns := make(map[string][]rtnetlink.LinkMessage)
	for ns, sp := range a.NetNS {
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

	// Start listening for HTTP connections.
	slog.Info("starting TC exporter", "listen-address", a.ListenAddres)
	if err := http.ListenAndServe(a.ListenAddres, mux); err != nil {
		slog.Error("cannot start TC exporter", "err", err.Error())
	}
	return nil
}

func main() {
	// CLI arguments parsing
	appCtx := kong.Parse(&cli,
		kong.Name("tc-exporter"),
		kong.Description("prometheus exporter for linux traffic control"),
		kong.UsageOnError(),
		kong.Vars{
			"version": "v0.8.0-rc1",
		},
		kong.Configuration(
			kongtoml.Loader,
			"/etc/tc-exporter/config.toml",
			"$HOME/.config/tc_exporter/config.toml",
			"./config.toml",
		),
	)

	fmt.Println(cli)

	var logLevel slog.Level
	switch cli.LogLevel {
	case "info":
		logLevel = slog.LevelInfo
	case "error":
		logLevel = slog.LevelError
	case "warn":
		logLevel = slog.LevelWarn
	case "debug":
		logLevel = slog.LevelDebug
	default:
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))
	slog.SetDefault(logger)

	err := appCtx.Run(logger)
	if err != nil {
		slog.Error("failed to run kong app", "error", err)
		os.Exit(5)
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
