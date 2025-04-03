package main

import (
	"log/slog"
	"net/http"
	"os"
	"fmt"

	"net/http/pprof"

	"github.com/alecthomas/kong"
	"github.com/spf13/viper"
	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/exporter-toolkit/web"
)

var cli App

// Config datasructure representing the configuration file
type Config struct {
	NetNS        map[string]NS
	Filters      tcexporter.FilterHolder `mapstructure:"filters"`
}

// NS holds a type alias so we can use it in the config file
type NS struct {
	Interfaces []string `name:"interfaces" mapstructure:"interfaces"`
}

// App holds are the
type App struct {
	Config        string `help:"location of the config path" name:"config-file"`
	LogLevel      string          `help:"slog based log level" default:"info" name:"log-level"`
	ListenAddres  string          `help:"address to listen on" default:":9704" name:"listen-address"`
	QdiscEnable   bool            `help:"enable the qdisc collector" negatable:"" default:"true" name:"collector-qdisc"`
	ClassEnable   bool            `help:"enable the class collector" negatable:"" default:"true" name:"collector-class"`
	CbqEnable     bool            `help:"enable the cbq collector" negatable:"" default:"false" name:"collector-cbq"`
	ChokeEnable   bool            `help:"enable the choke collector" negatable:"" default:"false" name:"collector-choke"`
	CodelEnable   bool            `help:"enable the codel collector" negatable:"" default:"false" name:"collector-codel"`
	FqEnable      bool            `help:"enable the fq collector" negatable:"" default:"false" name:"collector-fq"`
	FqcodelEnable bool            `help:"enable the fqcodel collector" negatable:"" default:"false" name:"collector-fqcodel"`
	HfscEnable    bool            `help:"enable the hfsc collector" negatable:"" default:"false" name:"collector-hfsc"`
	HtbEnable     bool            `help:"enable the htb collector" negatable:"" default:"false" name:"collector-htb"`
	PieEnable     bool            `help:"enable the pie collector" negatable:"" default:"false" name:"collector-pie"`
	RedEnable     bool            `help:"enable the red collector" negatable:"" default:"false" name:"collector-red"`
	SfbEnable     bool            `help:"enable the sfb collector" negatable:"" default:"false" name:"collector-sfb"`
	SfqEnable     bool            `help:"enable the sfq collector" negatable:"" default:"false" name:"collector-sfq"`
}

func (a *App) Run(logger *slog.Logger, cfg Config) error {
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

	enabledCollectors := map[string]bool{
		"qdisc":         a.QdiscEnable,
		"class":         a.ClassEnable,
		"cbq":           a.CbqEnable,
		"choke":         a.ChokeEnable,
		"codel":         a.CodelEnable,
		"fq":            a.FqEnable,
		"fq_codel":      a.FqcodelEnable,
		"hfsc_qdisc":    a.HfscEnable,
		"service_curve": a.HfscEnable,
		"htb":           a.HtbEnable,
		"pie":           a.PieEnable,
		"red":           a.RedEnable,
		"sfb":           a.SfbEnable,
		"sfq":           a.SfqEnable,
	}

	// initialise the collector with the configured subcollectors
	collector, err := tcexporter.NewTcCollector(netns, enabledCollectors, cfg.Filters, logger)
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
			"version": "v0.8.2",
		},
	)

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

	var config Config
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("toml") // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/etc/tc-exporter/")   // path to look for the config file in
	viper.AddConfigPath("$HOME/.tc-exporter")  // call multiple times to add many search paths
	viper.AddConfigPath(".")               // optionally look for config in the working directory
	viper.AddConfigPath(cli.Config)
	err := viper.ReadInConfig() // Find and read the config file
	if err != nil { // Handle errors reading the config file
		slog.Error("error reading config file", "err", err, "path", "test")
		os.Exit(10)
	}
	err = viper.Unmarshal(&config)
	if err != nil { // Handle errors reading the config file
		slog.Error("error unmarshalling config file", "err", err, "path", "test")
		os.Exit(11)
	}
	fmt.Println(config)

	err = appCtx.Run(logger, config)
	if err != nil {
		slog.Error("failed to run kong app", "error", err)
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
