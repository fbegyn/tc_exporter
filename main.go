package main

import (
	"fmt"
	"time"

	"github.com/influxdata/influxdb/client/v2"
	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	username = kingpin.Flag("user", "Username for influxdb").Default("admin").Short('u').String()
	password = kingpin.Flag("password", "Password for influxdb").Default("admin").Short('k').String()
	database = kingpin.Flag("database", "Influxdb database to use").Default("netlink").Short('d').String()
	port     = kingpin.Flag("port", "Influxdb data port").Default("8086").Short('p').Int16()
	host     = kingpin.Flag("host", "Influxdb server hostname").Default("localhost").Short('h').String()
	interval = kingpin.Flag("interval", "Interval to export the data").Default("15").Short('i').Int16()

	enableProm   = kingpin.Flag("prometheus", "Enable prometheus exporting").Default("false").Bool()
	promPort     = kingpin.Flag("promport", "Port on which the prometheus exporter runs").Default("9601").Short('P').Int16()
	enableInflux = kingpin.Flag("influx", "Enable influx exporting").Default("true").Short('I').Bool()

	interfaces = kingpin.Arg("network interface", "interface for which the exporter runs").Required().Strings()
)

func main() {
	// CLI arguments parsing
	kingpin.Version("v0.1.6")
	kingpin.Parse()
	// Configuring the logging
	customFormatter := new(logrus.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05"
	customFormatter.FullTimestamp = true
	logrus.SetFormatter(customFormatter)

	ticker := time.NewTicker(time.Duration(*interval) * time.Second)
	if *enableProm {
		logrus.Infoln("prometheus exporter enabled")
	}

	// Create Influxdb client
	dbclient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     fmt.Sprintf("http://%s:%d", *host, *port),
		Username: *username,
		Password: *password,
	})
	if err != nil {
		logrus.Fatalf("something went wrong with the influxdb connection: %v\n", err)
	}
	logrus.Infoln("influxdb client started")

	if *enableProm {
		go PromExporter(fmt.Sprintf(":%d", *promPort))
	}

	links := *interfaces

	for range ticker.C {
		for _, interf := range links {
			link, qdiscs, classes := GetData(interf)
			classes = FilterClass(&classes, "hfsc")
			if *enableProm {
				logrus.Infoln("starting prometheus export")
				go HandleProm(&link, &qdiscs, &classes)
			}
			if *enableInflux {
				logrus.Infoln("starting influxdb export")
				go WriteLink(dbclient, *database, link)
				go func() {
					WriteQdisc(dbclient, *database, &qdiscs)
				}()
				go func() {
					WriteClass(dbclient, *database, &classes)
				}()
			}
			logrus.Infoln("scrape completed")
		}
	}
}
