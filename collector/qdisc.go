package collector

import (
	"fmt"
	"net"
	"os"

	"github.com/florianl/go-tc"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	qdisclabels []string = []string{"host", "linkindex", "link", "type", "handle", "parent"}
)

type QdiscCollector struct {
	interf     string
	devID      uint32
	qdisc      tc.Object
	bytes      *prometheus.Desc
	packets    *prometheus.Desc
	bps        *prometheus.Desc
	pps        *prometheus.Desc
	backlog    *prometheus.Desc
	drops      *prometheus.Desc
	overlimits *prometheus.Desc
	qlen       *prometheus.Desc
}

func NewQdiscCollector(interf *net.Interface, qdisc tc.Object) (Collector, error) {
	return &QdiscCollector{
		interf: interf.Name,
		devID:  uint32(interf.Index),
		qdisc:  qdisc,
		bytes: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "qdisc", "bytes"),
			"Qdisc byte counter",
			qdisclabels, nil,
		),
		packets: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "qdisc", "packets"),
			"Qdisc packet counter",
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
		backlog: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "qdisc", "backlog"),
			"Qdisc queue backlog",
			qdisclabels, nil,
		),
		drops: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "qdisc", "drops"),
			"Qdisc queue drops",
			qdisclabels, nil,
		),
		overlimits: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "qdisc", "overlimits"),
			"Qdisc queue overlimits",
			qdisclabels, nil,
		),
		qlen: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "qdisc", "qlen"),
			"Qdisc queue length",
			qdisclabels, nil,
		),
	}, nil
}

func (qc *QdiscCollector) Update(ch chan<- prometheus.Metric) error {
	host, err := os.Hostname()
	if err != nil {
		return err
	}

	handleMaj, handleMin := HandleStr(qc.qdisc.Handle)
	parentMaj, parentMin := HandleStr(qc.qdisc.Parent)

	ch <- prometheus.MustNewConstMetric(
		qc.bytes,
		prometheus.CounterValue,
		float64(qc.qdisc.Stats.Bytes),
		host,
		fmt.Sprintf("%d", qc.devID),
		qc.interf,
		qc.qdisc.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		qc.packets,
		prometheus.CounterValue,
		float64(qc.qdisc.Stats.Packets),
		host,
		fmt.Sprintf("%d", qc.devID),
		qc.interf,
		qc.qdisc.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		qc.bps,
		prometheus.CounterValue,
		float64(qc.qdisc.Stats.Bps),
		host,
		fmt.Sprintf("%d", qc.devID),
		qc.interf,
		qc.qdisc.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		qc.pps,
		prometheus.CounterValue,
		float64(qc.qdisc.Stats.Pps),
		host,
		fmt.Sprintf("%d", qc.devID),
		qc.interf,
		qc.qdisc.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		qc.backlog,
		prometheus.CounterValue,
		float64(qc.qdisc.Stats.Backlog),
		host,
		fmt.Sprintf("%d", qc.devID),
		qc.interf,
		qc.qdisc.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		qc.drops,
		prometheus.CounterValue,
		float64(qc.qdisc.Stats.Drops),
		host,
		fmt.Sprintf("%d", qc.devID),
		qc.interf,
		qc.qdisc.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		qc.overlimits,
		prometheus.CounterValue,
		float64(qc.qdisc.Stats.Overlimits),
		host,
		fmt.Sprintf("%d", qc.devID),
		qc.interf,
		qc.qdisc.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		qc.qlen,
		prometheus.CounterValue,
		float64(qc.qdisc.Stats.Qlen),
		host,
		fmt.Sprintf("%d", qc.devID),
		qc.interf,
		qc.qdisc.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	return nil
}
