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
	sfbLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// SfbCollector is the object that will collect sfb qdisc data for the interface
type SfbCollector struct {
	logger slog.Logger
	netns  map[string][]rtnetlink.LinkMessage

	avgProbe    *prometheus.Desc
	bucketDrop  *prometheus.Desc
	childDrop   *prometheus.Desc
	earlyDrop   *prometheus.Desc
	marked      *prometheus.Desc
	maxProb     *prometheus.Desc
	maxQlen     *prometheus.Desc
	penaltyDrop *prometheus.Desc
	queueDrop   *prometheus.Desc
}

// NewSfbCollector create a new QdiscCollector given a network interface
func NewSfbCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (ObjectCollector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "sfb")
	log.Info("making sfb collector")

	return &SfbCollector{
		logger: *log,
		netns:  netns,
		avgProbe: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "sfb", "avg_probe"),
			"SFB avg probe xstat",
			sfbLabels, nil,
		),
		bucketDrop: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "sfb", "bucket_drop"),
			"SFB bucket drop xstat",
			sfbLabels, nil,
		),
		childDrop: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "sfb", "child_drop"),
			"SFB child drop xstat",
			sfbLabels, nil,
		),
		earlyDrop: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "sfb", "early_drop"),
			"SFB early drop xstat",
			sfbLabels, nil,
		),
		marked: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "sfb", "marked"),
			"SFB marked xstat",
			sfbLabels, nil,
		),
		maxProb: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "sfb", "max_prob"),
			"SFB max prob xstat",
			sfbLabels, nil,
		),
		maxQlen: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "sfb", "max_qlen"),
			"SFB max qlen xstat",
			sfbLabels, nil,
		),
		penaltyDrop: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "sfb", "penalty_drop"),
			"SFB penalty drop xstat",
			sfbLabels, nil,
		),
		queueDrop: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "sfb", "queue_drop"),
			"SFB queue drop xstat",
			sfbLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (col *SfbCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		col.avgProbe,
		col.bucketDrop,
		col.childDrop,
		col.earlyDrop,
		col.marked,
		col.maxProb,
		col.maxQlen,
		col.penaltyDrop,
		col.queueDrop,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (col *SfbCollector) Collect(ch chan<- prometheus.Metric) {
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
				if qd.Sfb == nil || qd.XStats == nil {
					continue
				}
				handleMaj, handleMin := HandleStr(qd.Handle)
				parentMaj, parentMin := HandleStr(qd.Parent)

				ch <- prometheus.MustNewConstMetric(
					col.avgProbe,
					prometheus.CounterValue,
					float64(qd.XStats.Sfb.AvgProb),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.bucketDrop,
					prometheus.CounterValue,
					float64(qd.XStats.Sfb.BucketDrop),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.childDrop,
					prometheus.CounterValue,
					float64(qd.XStats.Sfb.ChildDrop),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.earlyDrop,
					prometheus.CounterValue,
					float64(qd.XStats.Sfb.EarlyDrop),
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
					float64(qd.XStats.Sfb.Marked),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.maxProb,
					prometheus.CounterValue,
					float64(qd.XStats.Sfb.MaxProb),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.maxQlen,
					prometheus.CounterValue,
					float64(qd.XStats.Sfb.MaxQlen),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.penaltyDrop,
					prometheus.CounterValue,
					float64(qd.XStats.Sfb.PenaltyDrop),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.queueDrop,
					prometheus.CounterValue,
					float64(qd.XStats.Sfb.QueueDrop),
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
func (col *SfbCollector) CollectObject(ch chan<- prometheus.Metric, host, ns string, interf rtnetlink.LinkMessage, qd tc.Object) {
	if qd.XStats == nil {
		return
	}

	handleMaj, handleMin := HandleStr(qd.Handle)
	parentMaj, parentMin := HandleStr(qd.Parent)

	ch <- prometheus.MustNewConstMetric(
		col.avgProbe,
		prometheus.CounterValue,
		float64(qd.XStats.Sfb.AvgProb),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.bucketDrop,
		prometheus.CounterValue,
		float64(qd.XStats.Sfb.BucketDrop),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.childDrop,
		prometheus.CounterValue,
		float64(qd.XStats.Sfb.ChildDrop),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.earlyDrop,
		prometheus.CounterValue,
		float64(qd.XStats.Sfb.EarlyDrop),
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
		float64(qd.XStats.Sfb.Marked),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.maxProb,
		prometheus.CounterValue,
		float64(qd.XStats.Sfb.MaxProb),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.maxQlen,
		prometheus.CounterValue,
		float64(qd.XStats.Sfb.MaxQlen),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.penaltyDrop,
		prometheus.CounterValue,
		float64(qd.XStats.Sfb.PenaltyDrop),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.queueDrop,
		prometheus.CounterValue,
		float64(qd.XStats.Sfb.QueueDrop),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
}
