package main

import (
	"net/http"

	"github.com/povilasv/prommod"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	// enableProm = kingpin.Flag("prometheus", "Enable prometheus exporting").Default("true").Bool()
	promPort = kingpin.Flag("promport", "Port on which the prometheus exporter runs").Default("9601").Short('P').String()
	// enableInflux = kingpin.Flag("influx", "Enable influx exporting").Default("false").Short('I').Bool()

	// interfaces = kingpin.Arg("network interface", "interface for which the exporter runs").Required().Strings()
)

func main() {
	// CLI arguments parsing
	kingpin.Version("v0.1.7")
	kingpin.Parse()

	// Configuring the logging
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)

	_ = prometheus.Register(prommod.NewCollector("tc_exporter"))
	logrus.Infoln("prometheus exporter enabled")
	PromExporter()
	logrus.Fatal(http.ListenAndServe(":"+*promPort, nil))
}
