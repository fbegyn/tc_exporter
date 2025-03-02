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
	qdisclabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// QdiscCollector is the object that will collect Qdisc data for the interface
type QdiscCollector struct {
	logger slog.Logger
	netns  map[string][]rtnetlink.LinkMessage
	stats  stats
}

// NewQdiscCollector create a new QdiscCollector given a network interface
func NewQdiscCollector(netns map[string][]rtnetlink.LinkMessage, qlog *slog.Logger) (ObjectCollector, error) {
	// Setup logger for qdisc collector
	qlog = qlog.With("collector", "qdisc")
	qlog.Info("making qdisc collector")

	return &QdiscCollector{
		logger: *qlog,
		netns:  netns,
		stats: stats{
			bytes: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "qdisc", "bytes_total"),
				"Qdisc byte counter",
				qdisclabels, nil,
			),
			packets: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "qdisc", "packets_total"),
				"Qdisc packet counter",
				qdisclabels, nil,
			),
			drops: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "qdisc", "drops_total"),
				"Qdisc queue drops",
				qdisclabels, nil,
			),
			overlimits: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "qdisc", "overlimits_total"),
				"Qdisc queue overlimits",
				qdisclabels, nil,
			),
			bps: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "qdisc", "bps"),
				"Qdisc byte rate",
				qdisclabels, nil,
			),
			pps: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "qdisc", "pps"),
				"Qdisc packet rate",
				qdisclabels, nil,
			),
			qlen: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "qdisc", "qlen_total"),
				"Qdisc queue length",
				qdisclabels, nil,
			),
			backlog: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "qdisc", "backlog_total"),
				"Qdisc queue backlog",
				qdisclabels, nil,
			),
			requeues: prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "qdisc", "requeues_total"),
				"Qdisc queue requeues",
				qdisclabels, nil,
			),
		},
	}, nil
}

// Describe implements Collector
func (qc *QdiscCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		qc.stats.bytes,
		qc.stats.packets,
		qc.stats.drops,
		qc.stats.overlimits,
		qc.stats.bps,
		qc.stats.pps,
		qc.stats.qlen,
		qc.stats.backlog,
		qc.stats.requeues,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (qc *QdiscCollector) Collect(ch chan<- prometheus.Metric) {
	// fetch the host for useage later on
	host, err := os.Hostname()
	if err != nil {
		qc.logger.Error("failed to fetch hostname", "err", err)
	}

	// iterate through the netns and devices
	for ns, devices := range qc.netns {
		for _, interf := range devices {
			// fetch all the the qdisc for this interface
			qdiscs, err := getQdiscs(uint32(interf.Index), ns)
			if err != nil {
				qc.logger.Error("failed to get qdiscs", "interface", interf.Attributes.Name, "err", err)
			}

			// iterate through all the qdiscs and sent the data to the prometheus metric channel
			for _, qd := range qdiscs {
				handleMaj, handleMin := HandleStr(qd.Handle)
				parentMaj, parentMin := HandleStr(qd.Parent)
				var bytes, packets, drops, overlimits, qlen, backlog float64
				if qd.Stats2 != nil {
					bytes = float64(qd.Stats2.Bytes)
					packets = float64(qd.Stats2.Packets)
					drops = float64(qd.Stats2.Drops)
					overlimits = float64(qd.Stats2.Overlimits)
					qlen = float64(qd.Stats2.Qlen)
					backlog = float64(qd.Stats2.Backlog)
					ch <- prometheus.MustNewConstMetric(
						qc.stats.requeues,
						prometheus.CounterValue,
						float64(qd.Stats2.Requeues),
						host,
						ns,
						fmt.Sprintf("%d", interf.Index),
						interf.Attributes.Name,
						qd.Kind,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
				} else {
					qc.logger.Debug("stats2 struct is empty for this qdisc", "qdisc", qd)
				}
				if qd.Stats != nil {
					bytes = float64(qd.Stats.Bytes)
					packets = float64(qd.Stats.Packets)
					drops = float64(qd.Stats.Drops)
					overlimits = float64(qd.Stats.Overlimits)
					qlen = float64(qd.Stats.Qlen)
					backlog = float64(qd.Stats.Backlog)
					ch <- prometheus.MustNewConstMetric(
						qc.stats.bps,
						prometheus.GaugeValue,
						float64(qd.Stats.Bps),
						host,
						ns,
						fmt.Sprintf("%d", interf.Index),
						interf.Attributes.Name,
						qd.Kind,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
					ch <- prometheus.MustNewConstMetric(
						qc.stats.pps,
						prometheus.GaugeValue,
						float64(qd.Stats.Pps),
						host,
						ns,
						fmt.Sprintf("%d", interf.Index),
						interf.Attributes.Name,
						qd.Kind,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
				} else {
					qc.logger.Debug("stats struct is empty for this qdisc", "qdisc", qd)
				}
				if (qd.Stats != nil) || (qd.Stats2 != nil) {
					ch <- prometheus.MustNewConstMetric(
						qc.stats.bytes,
						prometheus.CounterValue,
						bytes,
						host,
						ns,
						fmt.Sprintf("%d", interf.Index),
						interf.Attributes.Name,
						qd.Kind,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
					ch <- prometheus.MustNewConstMetric(
						qc.stats.packets,
						prometheus.CounterValue,
						packets,
						host,
						ns,
						fmt.Sprintf("%d", interf.Index),
						interf.Attributes.Name,
						qd.Kind,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
					ch <- prometheus.MustNewConstMetric(
						qc.stats.backlog,
						prometheus.CounterValue,
						backlog,
						host,
						ns,
						fmt.Sprintf("%d", interf.Index),
						interf.Attributes.Name,
						qd.Kind,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
					ch <- prometheus.MustNewConstMetric(
						qc.stats.drops,
						prometheus.CounterValue,
						drops,
						host,
						ns,
						fmt.Sprintf("%d", interf.Index),
						interf.Attributes.Name,
						qd.Kind,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
					ch <- prometheus.MustNewConstMetric(
						qc.stats.overlimits,
						prometheus.CounterValue,
						overlimits,
						host,
						ns,
						fmt.Sprintf("%d", interf.Index),
						interf.Attributes.Name,
						qd.Kind,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
					ch <- prometheus.MustNewConstMetric(
						qc.stats.qlen,
						prometheus.CounterValue,
						qlen,
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
}

// CollectObject fetches and updates the data the collector is exporting
func (qc *QdiscCollector) CollectObject(ch chan<- prometheus.Metric, host, ns string, interf rtnetlink.LinkMessage, qd tc.Object) {
	handleMaj, handleMin := HandleStr(qd.Handle)
	parentMaj, parentMin := HandleStr(qd.Parent)

	var bytes, packets, drops, overlimits, qlen, backlog float64
	if qd.Stats2 != nil {
		bytes = float64(qd.Stats2.Bytes)
		packets = float64(qd.Stats2.Packets)
		drops = float64(qd.Stats2.Drops)
		overlimits = float64(qd.Stats2.Overlimits)
		qlen = float64(qd.Stats2.Qlen)
		backlog = float64(qd.Stats2.Backlog)
		ch <- prometheus.MustNewConstMetric(
			qc.stats.requeues,
			prometheus.CounterValue,
			float64(qd.Stats2.Requeues),
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
	} else {
		qc.logger.Debug("stats2 struct is empty for this qdisc", "qdisc", qd)
	}
	if qd.Stats != nil {
		bytes = float64(qd.Stats.Bytes)
		packets = float64(qd.Stats.Packets)
		drops = float64(qd.Stats.Drops)
		overlimits = float64(qd.Stats.Overlimits)
		qlen = float64(qd.Stats.Qlen)
		backlog = float64(qd.Stats.Backlog)
		ch <- prometheus.MustNewConstMetric(
			qc.stats.bps,
			prometheus.GaugeValue,
			float64(qd.Stats.Bps),
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.stats.pps,
			prometheus.GaugeValue,
			float64(qd.Stats.Pps),
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
	} else {
		qc.logger.Debug("stats struct is empty for this qdisc", "qdisc", qd)
	}
	if (qd.Stats != nil) || (qd.Stats2 != nil) {
		ch <- prometheus.MustNewConstMetric(
			qc.stats.bytes,
			prometheus.CounterValue,
			bytes,
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.stats.packets,
			prometheus.CounterValue,
			packets,
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.stats.backlog,
			prometheus.CounterValue,
			backlog,
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.stats.drops,
			prometheus.CounterValue,
			drops,
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.stats.overlimits,
			prometheus.CounterValue,
			overlimits,
			host,
			ns,
			fmt.Sprintf("%d", interf.Index),
			interf.Attributes.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.stats.qlen,
			prometheus.CounterValue,
			qlen,
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
