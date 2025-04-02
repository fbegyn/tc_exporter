package tccollector

import (
	"fmt"
	"log/slog"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	classlabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
	curvelabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// ClassCollector is the object that will collect Class data for the interface
// It is a basic reperesentation of the Stats and Stats2 struct of iproute
type ClassCollector struct {
	logger slog.Logger
	netns  map[string][]rtnetlink.LinkMessage
	stats  stats
}

// NewClassCollector create a new ClassCollector given a network interface
func NewClassCollector(netns map[string][]rtnetlink.LinkMessage, clog *slog.Logger) (ObjectCollector, error) {
	// Setup logger for the class collector
	clog = clog.With("collector", "class")
	clog.Info("making class collector")

	return &ClassCollector{
		logger: *clog,
		netns:  netns,
		stats: stats{
			bytes: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "class", "bytes_total"),
				"Class counter",
				classlabels, nil,
			),
			packets: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "class", "packets_total"),
				"Class packet counter",
				classlabels, nil,
			),
			drops: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "class", "drops_total"),
				"Class queue drops",
				classlabels, nil,
			),
			overlimits: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "class", "overlimits_total"),
				"Class queue overlimits",
				classlabels, nil,
			),
			bps: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "class", "bps"),
				"Class byte rate",
				classlabels, nil,
			),
			pps: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "class", "pps"),
				"Class packet rate",
				classlabels, nil,
			),
			qlen: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "class", "qlen_total"),
				"Class queue length",
				classlabels, nil,
			),
			backlog: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "class", "backlog_total"),
				"Class queue backlog",
				classlabels, nil,
			),
			requeues: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "class", "requeque_total"),
				"Class requeque counter",
				classlabels, nil,
			),
		},
	}, nil
}

// Describe implements Collector
func (cc *ClassCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		cc.stats.bytes,
		cc.stats.packets,
		cc.stats.drops,
		cc.stats.overlimits,
		cc.stats.bps,
		cc.stats.pps,
		cc.stats.qlen,
		cc.stats.backlog,
		cc.stats.requeues,
	}

	for _, d := range ds {
		ch <- d
	}
}

// CollectObject fetches and updates the data the collector is exporting
func (cc *ClassCollector) CollectObject(ch chan<- prometheus.Metric, host, ns string, interf rtnetlink.LinkMessage, cl tc.Object) {
	handleMaj, handleMin := HandleStr(cl.Handle)
	parentMaj, parentMin := HandleStr(cl.Parent)

	var bytes, packets, drops, overlimits, qlen, backlog float64
	if cl.Stats2 != nil {
		bytes = float64(cl.Stats2.Bytes)
		packets = float64(cl.Stats2.Packets)
		drops = float64(cl.Stats2.Drops)
		overlimits = float64(cl.Stats2.Overlimits)
		qlen = float64(cl.Stats2.Qlen)
		backlog = float64(cl.Stats2.Backlog)
		ch <- prometheus.MustNewConstMetric(
			cc.stats.requeues,
			prometheus.CounterValue,
			float64(cl.Stats2.Requeues),
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			cl.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
	} else {
		cc.logger.Debug("stats2 struct is empty for this class", "class", cl)
	}
	if cl.Stats != nil {
		bytes = float64(cl.Stats.Bytes)
		packets = float64(cl.Stats.Packets)
		drops = float64(cl.Stats.Drops)
		overlimits = float64(cl.Stats.Overlimits)
		qlen = float64(cl.Stats.Qlen)
		backlog = float64(cl.Stats.Backlog)
		ch <- prometheus.MustNewConstMetric(
			cc.stats.bps,
			prometheus.GaugeValue,
			float64(cl.Stats.Bps),
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			cl.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			cc.stats.pps,
			prometheus.GaugeValue,
			float64(cl.Stats.Pps),
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			cl.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
	} else {
		cc.logger.Debug("stats struct is empty for this class", "class", cl)
	}
	if (cl.Stats != nil) || (cl.Stats2 != nil) {
		ch <- prometheus.MustNewConstMetric(
			cc.stats.bytes,
			prometheus.CounterValue,
			bytes,
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			cl.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			cc.stats.packets,
			prometheus.CounterValue,
			packets,
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			cl.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			cc.stats.backlog,
			prometheus.CounterValue,
			backlog,
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			cl.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			cc.stats.drops,
			prometheus.CounterValue,
			drops,
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			cl.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			cc.stats.overlimits,
			prometheus.CounterValue,
			overlimits,
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			cl.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			cc.stats.qlen,
			prometheus.CounterValue,
			qlen,
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			cl.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
	}
}
