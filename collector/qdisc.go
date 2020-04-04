package tccollector

import (
	"fmt"
	"net"
	"os"

	"github.com/florianl/go-tc"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	qdisclabels []string = []string{"host", "linkindex", "link", "type", "handle", "parent"}
)

// QdiscCollector is the object that will collect Qdisc data for the interface
type QdiscCollector struct {
	logger     log.Logger
	interf     *net.Interface
	sock       *tc.Tc
	bytes      *prometheus.Desc
	packets    *prometheus.Desc
	bps        *prometheus.Desc
	pps        *prometheus.Desc
	backlog    *prometheus.Desc
	drops      *prometheus.Desc
	overlimits *prometheus.Desc
	qlen       *prometheus.Desc
}

// NewQdiscCollector create a new QdiscCollector given a network interface
func NewQdiscCollector(interf *net.Interface, qlog log.Logger) (prometheus.Collector, error) {
	// Setup logger for qdisc collector
	qlog = log.With(qlog, "collector", "qdisc")
	qlog.Log("msg", "making qdisc collector", "inteface", interf.Name)

	// Create socket for interface to get qdiscs from
	rtnl, err := tc.Open(&tc.Config{})
	if err != nil {
		qlog.Log("msg", "could not open rtnetlink socket", "err", err)
		return nil, err
	}
	defer func() {
		if err := rtnl.Close(); err != nil {
			qlog.Log("msg", "could not close rtnetlink socket", "err", err)
		}
	}()

	return &QdiscCollector{
		logger: qlog,
		interf: interf,
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
			prometheus.BuildFQName(namespace, "qdisc", "backlog_total"),
			"Qdisc queue backlog",
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
		qlen: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "qdisc", "qlen_total"),
			"Qdisc queue length",
			qdisclabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (qc *QdiscCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		qc.backlog,
		qc.bps,
		qc.bytes,
		qc.packets,
		qc.drops,
		qc.overlimits,
		qc.pps,
		qc.qlen,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (qc *QdiscCollector) Collect(ch chan<- prometheus.Metric) {
	host, err := os.Hostname()
	if err != nil {
		qc.logger.Log("msg", "failed to fetch hostname", "err", err)
	}

	qdiscs, err := getQdiscs(uint32(qc.interf.Index))
	if err != nil {
		qc.logger.Log("msg", "failed to get qdiscs", "interface", qc.interf.Name, "err", err)
	}

	for _, qd := range qdiscs {

		handleMaj, handleMin := HandleStr(qd.Handle)
		parentMaj, parentMin := HandleStr(qd.Parent)

		ch <- prometheus.MustNewConstMetric(
			qc.bytes,
			prometheus.CounterValue,
			float64(qd.Stats.Bytes),
			host,
			fmt.Sprintf("%d", qc.interf.Index),
			qc.interf.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.packets,
			prometheus.CounterValue,
			float64(qd.Stats.Packets),
			host,
			fmt.Sprintf("%d", qc.interf.Index),
			qc.interf.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.bps,
			prometheus.GaugeValue,
			float64(qd.Stats.Bps),
			host,
			fmt.Sprintf("%d", qc.interf.Index),
			qc.interf.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.pps,
			prometheus.GaugeValue,
			float64(qd.Stats.Pps),
			host,
			fmt.Sprintf("%d", qc.interf.Index),
			qc.interf.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.backlog,
			prometheus.CounterValue,
			float64(qd.Stats.Backlog),
			host,
			fmt.Sprintf("%d", qc.interf.Index),
			qc.interf.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.drops,
			prometheus.CounterValue,
			float64(qd.Stats.Drops),
			host,
			fmt.Sprintf("%d", qc.interf.Index),
			qc.interf.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.overlimits,
			prometheus.CounterValue,
			float64(qd.Stats.Overlimits),
			host,
			fmt.Sprintf("%d", qc.interf.Index),
			qc.interf.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
		ch <- prometheus.MustNewConstMetric(
			qc.qlen,
			prometheus.CounterValue,
			float64(qd.Stats.Qlen),
			host,
			fmt.Sprintf("%d", qc.interf.Index),
			qc.interf.Name,
			qd.Kind,
			fmt.Sprintf("%x:%x", handleMaj, handleMin),
			fmt.Sprintf("%x:%x", parentMaj, parentMin),
		)
	}
}

func getQdiscs(devid uint32) ([]tc.Object, error) {
	// Create socket for interface to get qdiscs from
	sock, err := tc.Open(&tc.Config{})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := sock.Close(); err != nil {
		}
	}()

	qdiscs, err := sock.Qdisc().Get()
	if err != nil {
		return nil, err
	}
	var qd []tc.Object
	for _, qdisc := range qdiscs {
		if qdisc.Ifindex == devid {
			qd = append(qd, qdisc)
		}
	}
	return qd, nil
}
