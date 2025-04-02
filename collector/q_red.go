package tccollector

import (
	"fmt"
	"log/slog"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	redLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// RedCollector is the object that will collect RED qdisc data for the interface
type RedCollector struct {
	logger slog.Logger
	netns  map[string][]rtnetlink.LinkMessage

	early  *prometheus.Desc
	marked *prometheus.Desc
	other  *prometheus.Desc
	pDrop  *prometheus.Desc
}

// NewRedCollector create a new QdiscCollector given a network interface
func NewRedCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (ObjectCollector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "red")
	log.Info("making red collector")

	return &RedCollector{
		logger: *log,
		netns:  netns,
		early: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "red", "early"),
			"RED early xstat",
			redLabels, nil,
		),
		marked: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "red", "marked"),
			"RED marked xstat",
			redLabels, nil,
		),
		other: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "red", "other"),
			"RED other xstat",
			redLabels, nil,
		),
		pDrop: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "red", "pdrop"),
			"RED pdrop xstat",
			redLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (col *RedCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		col.early,
		col.marked,
		col.other,
		col.pDrop,
	}

	for _, d := range ds {
		ch <- d
	}
}

// CollectObject fetches and updates the data the collector is exporting
func (col *RedCollector) CollectObject(ch chan<- prometheus.Metric, host, ns string, interf rtnetlink.LinkMessage, qd tc.Object) {
	handleMaj, handleMin := HandleStr(qd.Handle)
	parentMaj, parentMin := HandleStr(qd.Parent)

	ch <- prometheus.MustNewConstMetric(
		col.early,
		prometheus.CounterValue,
		float64(qd.XStats.Red.Early),
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
		float64(qd.XStats.Red.Marked),
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
		float64(qd.XStats.Red.Other),
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
		float64(qd.XStats.Red.PDrop),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
}
