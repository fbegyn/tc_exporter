package main

import (
	"fmt"
	"net"
	"net/http"
	"os"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/go-kit/kit/log"
	"github.com/jsimonetti/rtnetlink"
	"github.com/mdlayher/netlink"
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

	var cf Config
	cf.ListenAddres = viper.GetString("listen-address")
	err := viper.Unmarshal(&cf)
	if err != nil {
		logger.Log("level", "ERROR", "msg", "failed to read config file", "error", err)
	}
	logger.Log("msg", "succesfully read config file")

	netns := make(map[int][]*net.Interface)
	for ns, space := range cf.NetNS {
		for _, device := range space.Interfaces {
			con, err := getInterfaceInNS(device, ns)
			defer con.Close()
			interf, err := net.InterfaceByName(device)
			if err != nil {
				logger.Log("err", "could not get interface by name", "interface", device)
			}
			if interf == nil {
				logger.Log("warn", "interface does not exist, SKIPPING IT!", "interface", device)
				continue
			}
			logger.Log("msg", "add interface to ns", "interface", device, "netns", ns)
			netns[ns] = append(netns[ns], interf)
		}
	}

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

func getInterfaceInNS(interf string, ns int) (*rtnetlink.Conn, error) {
	fmt.Printf("Dialing: netns %d and device %s\n", ns, interf)
	con, err := rtnetlink.Dial(&netlink.Config{
		Groups:              0,
		NetNS:               ns,
		DisableNSLockThread: false,
	})
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	link, _ := con.Link.List()
	for _, l := range link {
		fmt.Println(l.Attributes)
	}

	return con, nil
}
