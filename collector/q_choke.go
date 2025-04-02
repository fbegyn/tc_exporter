package tccollector

import (
	"fmt"
	"log/slog"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	chokeLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// ChokeCollector is the object that will collect choke qdisc data for the interface
type ChokeCollector struct {
	logger  slog.Logger
	netns   map[string][]rtnetlink.LinkMessage
	early   *prometheus.Desc
	marked  *prometheus.Desc
	matched *prometheus.Desc
	other   *prometheus.Desc
	pDrop   *prometheus.Desc
}

// NewChokeCollector create a new QdiscCollector given a network interface
func NewChokeCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (ObjectCollector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "choke")
	log.Info("making choke collector")

	return &ChokeCollector{
		logger: *log,
		netns:  netns,
		early: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "choke", "early"),
			"Choke early xstat",
			chokeLabels, nil,
		),
		marked: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "choke", "marked"),
			"Choke marked xstat",
			chokeLabels, nil,
		),
		matched: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "choke", "matched"),
			"Choke matched xstat",
			chokeLabels, nil,
		),
		other: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "choke", "other"),
			"Choke other xstat",
			chokeLabels, nil,
		),
		pDrop: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "choke", "pdrop"),
			"Choke pdrop xstat",
			chokeLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (col *ChokeCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		col.early,
		col.marked,
		col.matched,
		col.other,
		col.pDrop,
	}

	for _, d := range ds {
		ch <- d
	}
}

// CollectObject fetches and updates the data the collector is exporting
// func (col *CbqCollector) Collect(ch chan<- prometheus.Metric) {
func (col *ChokeCollector) CollectObject(ch chan<- prometheus.Metric, host, ns string, interf rtnetlink.LinkMessage, qd tc.Object) {
	handleMaj, handleMin := HandleStr(qd.Handle)
	parentMaj, parentMin := HandleStr(qd.Parent)

	ch <- prometheus.MustNewConstMetric(
		col.early,
		prometheus.CounterValue,
		float64(qd.XStats.Choke.Early),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.marked,
		prometheus.CounterValue,
		float64(qd.XStats.Choke.Marked),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.matched,
		prometheus.CounterValue,
		float64(qd.XStats.Choke.Matched),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.other,
		prometheus.CounterValue,
		float64(qd.XStats.Choke.Other),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.pDrop,
		prometheus.CounterValue,
		float64(qd.XStats.Choke.PDrop),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
}
