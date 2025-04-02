package tccollector

import (
	"fmt"
	"log/slog"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	sfqLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// SfqCollector is the object that will collect sfq qdisc data for the interface
type SfqCollector struct {
	logger slog.Logger
	netns  map[string][]rtnetlink.LinkMessage

	allot *prometheus.Desc
}

// NewSfqCollector create a new QdiscCollector given a network interface
func NewSfqCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (ObjectCollector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "sfq")
	log.Info("making sfq collector")

	return &SfqCollector{
		logger: *log,
		netns:  netns,
		allot: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "sfq", "allot"),
			"SFQ allot xstat",
			sfqLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (col *SfqCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{col.allot}

	for _, d := range ds {
		ch <- d
	}
}

// CollectObject fetches and updates the data the collector is exporting
func (col *SfqCollector) CollectObject(ch chan<- prometheus.Metric, host, ns string, interf rtnetlink.LinkMessage, qd tc.Object) {
	handleMaj, handleMin := HandleStr(qd.Handle)
	parentMaj, parentMin := HandleStr(qd.Parent)

	ch <- prometheus.MustNewConstMetric(
		col.allot,
		prometheus.CounterValue,
		float64(qd.XStats.Sfq.Allot),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
}
