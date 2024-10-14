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
	cbqLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// CBQCollector is the object that will collect CBQ qdisc data for the interface
type CBQCollector struct {
	logger     slog.Logger
	netns      map[string][]rtnetlink.LinkMessage
	AvgIdle  *prometheus.Desc
	Borrows  *prometheus.Desc
	Overactions  *prometheus.Desc
	Undertime  *prometheus.Desc
}

// NewCBQCollector create a new QdiscCollector given a network interface
func NewCBQCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (prometheus.Collector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "cbq")
	log.Debug("making cbq collector")

	return &CBQCollector{
		logger: *log,
		netns:  netns,
		AvgIdle: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cbq", "avg_idle"),
			"CBQ avg idle xstat",
			cbqLabels, nil,
		),
		Borrows: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cbq", "borrows"),
			"CBQ borrows xstat",
			cbqLabels, nil,
		),
		Overactions: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cbq", "overactions"),
			"CBQ overactions xstat",
			cbqLabels, nil,
		),
		Undertime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cbq", "under_time"),
			"CBQ under time xstat",
			cbqLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (col *CBQCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (col *CBQCollector) Collect(ch chan<- prometheus.Metric) {
	// fetch the host for useage later on
	host, err := os.Hostname()
	if err != nil {
		col.logger.Error("failed to fetch hostname", "err", err)
	}

	// iterate through the netns and devices
	for ns, devices := range col.netns {
		for _, interf := range devices {
			// fetch all the the qdisc for this interface
			qdiscs, err := getFQQdiscs(uint32(interf.Index), ns)
			if err != nil {
				col.logger.Error("failed to get qdiscs", "interface", interf.Attributes.Name, "err", err)
			}

			// iterate through all the qdiscs and sent the data to the prometheus metric channel
			for _, qd := range qdiscs {
				handleMaj, handleMin := HandleStr(qd.Handle)
				parentMaj, parentMin := HandleStr(qd.Parent)

				ch <- prometheus.MustNewConstMetric(
					col.AvgIdle,
					prometheus.CounterValue,
					float64(qd.XStats.Cbq.AvgIdle),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.Borrows,
					prometheus.CounterValue,
					float64(qd.XStats.Cbq.Borrows),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.Overactions,
					prometheus.CounterValue,
					float64(qd.XStats.Cbq.Overactions),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.Undertime,
					prometheus.CounterValue,
					float64(qd.XStats.Cbq.Undertime),
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
