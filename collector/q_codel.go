package tccollector

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	codelLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// CodelCollector is the object that will collect codel qdisc data for the interface
type CodelCollector struct {
	logger        slog.Logger
	netns         map[string][]rtnetlink.LinkMessage
	ceMark        *prometheus.Desc
	count         *prometheus.Desc
	dropNext      *prometheus.Desc
	dropOverlimit *prometheus.Desc
	dropping      *prometheus.Desc
	ecnMark       *prometheus.Desc
	lDelay        *prometheus.Desc
	lastCount     *prometheus.Desc
	maxPacket     *prometheus.Desc
}

// NewCodelCollector create a new QdiscCollector given a network interface
func NewCodelCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (prometheus.Collector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "codel")
	log.Info("making codel collector")

	return &CodelCollector{
		logger: *log,
		netns:  netns,
		ceMark: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "codel", "ce_mark"),
			"Codel CE mark xstat",
			codelLabels, nil,
		),
		count: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "codel", "count"),
			"Codel count xstat",
			codelLabels, nil,
		),
		dropNext: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "codel", "drop_next"),
			"Codel drop next xstat",
			codelLabels, nil,
		),
		dropOverlimit: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "codel", "drop_overlimit"),
			"Codel drop overlimit xstat",
			codelLabels, nil,
		),
		dropping: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "codel", "dropping"),
			"Codel dropping xstat",
			codelLabels, nil,
		),
		ecnMark: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "codel", "ecn_mark"),
			"Codel ecn mark xstat",
			codelLabels, nil,
		),
		lDelay: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "codel", "ldelay"),
			"Codel ldelay xstat",
			codelLabels, nil,
		),
		lastCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "codel", "last_count"),
			"Codel last count xstat",
			codelLabels, nil,
		),
		maxPacket: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "codel", "max_packet"),
			"Codel max packet xstat",
			codelLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (col *CodelCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		col.ceMark,
		col.count,
		col.dropNext,
		col.dropOverlimit,
		col.dropping,
		col.ecnMark,
		col.lDelay,
		col.lastCount,
		col.maxPacket,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (col *CodelCollector) Collect(ch chan<- prometheus.Metric) {
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
				if qd.Choke == nil {
					continue
				}
				handleMaj, handleMin := HandleStr(qd.Handle)
				parentMaj, parentMin := HandleStr(qd.Parent)

				ch <- prometheus.MustNewConstMetric(
					col.ceMark,
					prometheus.CounterValue,
					float64(qd.XStats.Codel.CeMark),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.count,
					prometheus.CounterValue,
					float64(qd.XStats.Codel.Count),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.dropNext,
					prometheus.CounterValue,
					float64(qd.XStats.Codel.DropNext),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.dropOverlimit,
					prometheus.CounterValue,
					float64(qd.XStats.Codel.DropOverlimit),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.dropping,
					prometheus.CounterValue,
					float64(qd.XStats.Codel.Dropping),
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
					float64(qd.XStats.Codel.EcnMark),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.lDelay,
					prometheus.CounterValue,
					float64(qd.XStats.Codel.LDelay),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.lastCount,
					prometheus.CounterValue,
					float64(qd.XStats.Codel.LastCount),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.maxPacket,
					prometheus.CounterValue,
					float64(qd.XStats.Codel.MaxPacket),
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
