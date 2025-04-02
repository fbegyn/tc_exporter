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
	pieLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// PieCollector is the object that will collect pie qdisc data for the interface
type PieCollector struct {
	logger slog.Logger
	netns  map[string][]rtnetlink.LinkMessage

	avgDqRate *prometheus.Desc
	delay     *prometheus.Desc
	dropped   *prometheus.Desc
	ecnMark   *prometheus.Desc
	maxq      *prometheus.Desc
	overlimit *prometheus.Desc
	packetsIn *prometheus.Desc
	prob      *prometheus.Desc
}

// NewPieCollector create a new QdiscCollector given a network interface
func NewPieCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (ObjectCollector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "pie")
	log.Info("making pie collector")

	return &PieCollector{
		logger: *log,
		netns:  netns,
		avgDqRate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pie", "avg_dq_rate"),
			"PIE avgdqrate xstat",
			pieLabels, nil,
		),
		delay: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pie", "delay"),
			"PIE delay xstat",
			pieLabels, nil,
		),
		dropped: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pie", "dropped"),
			"PIE dropped",
			pieLabels, nil,
		),
		ecnMark: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pie", "ecn_mark"),
			"PIE ecn mark xstat",
			pieLabels, nil,
		),
		maxq: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pie", "maxq"),
			"PIE maxq xstat",
			pieLabels, nil,
		),
		overlimit: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pie", "overlimit"),
			"PIE overlimit xstat",
			pieLabels, nil,
		),
		packetsIn: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pie", "packets_in"),
			"PIE packets in xstat",
			pieLabels, nil,
		),
		prob: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "pie", "prob"),
			"PIE prob xstat",
			pieLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (col *PieCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		col.avgDqRate,
		col.delay,
		col.dropped,
		col.ecnMark,
		col.maxq,
		col.overlimit,
		col.packetsIn,
		col.prob,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (col *PieCollector) Collect(ch chan<- prometheus.Metric) {
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
				if qd.Pie == nil || qd.XStats == nil {
					continue
				}
				handleMaj, handleMin := HandleStr(qd.Handle)
				parentMaj, parentMin := HandleStr(qd.Parent)

				ch <- prometheus.MustNewConstMetric(
					col.avgDqRate,
					prometheus.CounterValue,
					float64(qd.XStats.Pie.AvgDqRate),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.delay,
					prometheus.CounterValue,
					float64(qd.XStats.Pie.Delay),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.dropped,
					prometheus.CounterValue,
					float64(qd.XStats.Pie.Dropped),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.ecnMark,
					prometheus.CounterValue,
					float64(qd.XStats.Pie.EcnMark),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.maxq,
					prometheus.CounterValue,
					float64(qd.XStats.Pie.Maxq),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.overlimit,
					prometheus.CounterValue,
					float64(qd.XStats.Pie.Overlimit),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.packetsIn,
					prometheus.CounterValue,
					float64(qd.XStats.Pie.PacketsIn),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.prob,
					prometheus.CounterValue,
					float64(qd.XStats.Pie.Prob),
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
func (col *PieCollector) CollectObject(ch chan<- prometheus.Metric, host, ns string, interf rtnetlink.LinkMessage, qd tc.Object) {
	if qd.XStats == nil {
		return
	}

	handleMaj, handleMin := HandleStr(qd.Handle)
	parentMaj, parentMin := HandleStr(qd.Parent)

	ch <- prometheus.MustNewConstMetric(
		col.avgDqRate,
		prometheus.CounterValue,
		float64(qd.XStats.Pie.AvgDqRate),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.delay,
		prometheus.CounterValue,
		float64(qd.XStats.Pie.Delay),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.dropped,
		prometheus.CounterValue,
		float64(qd.XStats.Pie.Dropped),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.ecnMark,
		prometheus.CounterValue,
		float64(qd.XStats.Pie.EcnMark),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.maxq,
		prometheus.CounterValue,
		float64(qd.XStats.Pie.Maxq),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.overlimit,
		prometheus.CounterValue,
		float64(qd.XStats.Pie.Overlimit),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.packetsIn,
		prometheus.CounterValue,
		float64(qd.XStats.Pie.PacketsIn),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.prob,
		prometheus.CounterValue,
		float64(qd.XStats.Pie.Prob),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
}
