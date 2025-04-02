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

// CbqCollector is the object that will collect CBQ qdisc data for the interface
type CbqCollector struct {
	logger      slog.Logger
	netns       map[string][]rtnetlink.LinkMessage
	avgIdle     *prometheus.Desc
	borrows     *prometheus.Desc
	overactions *prometheus.Desc
	undertime   *prometheus.Desc
}

// NewCbqCollector create a new QdiscCollector given a network interface
func NewCbqCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (ObjectCollector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "cbq")
	log.Debug("making cbq collector")

	return &CbqCollector{
		logger: *log,
		netns:  netns,
		avgIdle: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cbq", "avg_idle"),
			"CBQ avg idle xstat",
			cbqLabels, nil,
		),
		borrows: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cbq", "borrows"),
			"CBQ borrows xstat",
			cbqLabels, nil,
		),
		overactions: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cbq", "overactions"),
			"CBQ overactions xstat",
			cbqLabels, nil,
		),
		undertime: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cbq", "under_time"),
			"CBQ under time xstat",
			cbqLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (col *CbqCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		col.avgIdle,
		col.borrows,
		col.overactions,
		col.undertime,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
// func (col *CbqCollector) Collect(ch chan<- prometheus.Metric) {
func (col *CbqCollector) Collect(ch chan<- prometheus.Metric) {
	// fetch the host for useage later on
	host, err := os.Hostname()
	if err != nil {
		col.logger.Error("failed to fetch hostname", "err", err)
	}

	// iterate through the netns and devices
	for ns, devices := range col.netns {
		for _, interf := range devices {
			// fetch all the the qdisc for this interface
			qdiscs, err := getQdiscs(uint32(interf.Index), ns)
			if err != nil {
				col.logger.Error("failed to get qdiscs", "interface", interf.Attributes.Name, "err", err)
			}

			// iterate through all the qdiscs and sent the data to the prometheus metric channel
			for _, qd := range qdiscs {
				if qd.Cbq == nil || qd.XStats == nil {
					continue
				}
				handleMaj, handleMin := HandleStr(qd.Handle)
				parentMaj, parentMin := HandleStr(qd.Parent)

				ch <- prometheus.MustNewConstMetric(
					col.avgIdle,
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
					col.borrows,
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
					col.overactions,
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
					col.undertime,
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

// CollectObject fetches and updates the data the collector is exporting
// func (col *CbqCollector) Collect(ch chan<- prometheus.Metric) {
func (col *CbqCollector) CollectObject(ch chan<- prometheus.Metric, host, ns string, interf rtnetlink.LinkMessage, qd tc.Object) {
	if qd.XStats == nil {
		return
	}

	handleMaj, handleMin := HandleStr(qd.Handle)
	parentMaj, parentMin := HandleStr(qd.Parent)

	ch <- prometheus.MustNewConstMetric(
		col.avgIdle,
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
		col.borrows,
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
		col.overactions,
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
		col.undertime,
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
