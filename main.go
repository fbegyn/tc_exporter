package main

import (
  "time"
	"fmt"
	"github.com/influxdata/influxdb/client/v2"
  "github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

var (
	systemLink string
)

var (
	username   = kingpin.Flag("user", "Username for influxdb").Default("admin").Short('u').String()
	password   = kingpin.Flag("password", "Password for influxdb").Default("admin").Short('k').String()
	database      = kingpin.Flag("database", "Influxdb database to use").Default("netlink").Short('d').String()
	port       = kingpin.Flag("port", "Influxdb data port").Default("8080").Short('p').Int16()
	host       = kingpin.Flag("host", "Influxdb server hostname").Default("localhost").Short('h').String()
	interval       = kingpin.Flag("interval", "Interval to export the data").Default("15").Short('i').Int16()

  interf   = kingpin.Arg("network interface", "interface for which the exporter runs").Required().String()
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

	// Create Influxdb client
	dbclient, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     fmt.Sprintf("http://%s:%d", *host, *port),
		Username: *username,
		Password: *password,
	})
	if err != nil {
    logrus.Fatalf("something went wrong with the influx connection: %v\n", err)
	}
  logrus.Infoln(dbclient)

  ticker := time.NewTicker(time.Duration(*interval) * time.Second)
  for range ticker.C {
    link, qdiscs, classes := GetData(*interf)
    go WriteLink(dbclient, *database, link)
    logrus.Infoln(qdiscs)
    go func(){
      for _, qdisc := range qdiscs{
        WriteQdisc(dbclient, *database, qdisc)
      }
    }()
    logrus.Infoln(classes)
    go func(){
      for _, class := range classes{
        WriteClass(dbclient, *database, class)
      }
    }()
  }
}
