package tccollector

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	redLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// FqCollector is the object that will collect FQ qdisc data for the interface
type RedCollector struct {
	logger     slog.Logger
	netns      map[string][]rtnetlink.LinkMessage
}

// NewFqCollector create a new QdiscCollector given a network interface
func NewRedCollector(netns map[string][]rtnetlink.LinkMessage, fqcodellog *slog.Logger) (prometheus.Collector, error) {
	// Setup logger for qdisc collector
	fqcodellog = fqcodellog.With("collector", "fq_codel")
	fqcodellog.Info("making qdisc collector")

	return &FqCollector{
		logger: *fqcodellog,
		netns:  netns,
		gcFlows: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq_codel", "gc_flows"),
			"FQ gc flow counter",
			redLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (qc *RedCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (qc *RedCollector) Collect(ch chan<- prometheus.Metric) {
	// fetch the host for useage later on
	host, err := os.Hostname()
	if err != nil {
		qc.logger.Error("failed to fetch hostname", "err", err)
	}

	// iterate through the netns and devices
	for ns, devices := range qc.netns {
		for _, interf := range devices {
			// fetch all the the qdisc for this interface
			qdiscs, err := getFQQdiscs(uint32(interf.Index), ns)
			if err != nil {
				qc.logger.Error("failed to get qdiscs", "interface", interf.Attributes.Name, "err", err)
			}

			// iterate through all the qdiscs and sent the data to the prometheus metric channel
			for _, qd := range qdiscs {
				handleMaj, handleMin := HandleStr(qd.Handle)
				parentMaj, parentMin := HandleStr(qd.Parent)

				ch <- prometheus.MustNewConstMetric(
					qc.type,
					prometheus.CounterValue,
					float64(qd.XStats.Red.Type),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
			}
		}
	}
}

// getQdiscs fetches all qdiscs for a pecified interface in the netns
func getFQCodelQdiscs(devid uint32, ns string) ([]tc.Object, error) {
	sock, err := GetTcConn(ns)
	if err != nil {
		return nil, err
	}
	defer sock.Close()
	qdiscs, err := sock.Qdisc().Get()
	if err != nil {
		return nil, err
	}
	var qd []tc.Object
	for _, qdisc := range qdiscs {
		if qdisc.Ifindex == devid {
			if qdisc.Kind == "fq" {
				qd = append(qd, qdisc)
			}
		}
	}
	return qd, nil
}
