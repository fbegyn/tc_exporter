package tccollector

import (
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
)

var (
	classlabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
	curvelabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// ClassCollector is the object that will collect Class data for the interface
// It is a basic reperesentation of the Stats and Stats2 struct of iproute
type ClassCollector struct {
	logger     slog.Logger
	netns      map[string][]rtnetlink.LinkMessage
	bytes      *prometheus.Desc
	packets    *prometheus.Desc
	bps        *prometheus.Desc
	pps        *prometheus.Desc
	backlog    *prometheus.Desc
	drops      *prometheus.Desc
	overlimits *prometheus.Desc
	qlen       *prometheus.Desc
	requeues   *prometheus.Desc
}

// NewClassCollector create a new ClassCollector given a network interface
func NewClassCollector(netns map[string][]rtnetlink.LinkMessage, clog *slog.Logger) (prometheus.Collector, error) {
	// Setup logger for the class collector
	clog = clog.With("collector", "class")
	clog.Info("making class collector")

	return &ClassCollector{
		logger: *clog,
		netns:  netns,
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
		backlog: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "backlog_total"),
			"Class queue backlog",
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
		qlen: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "qlen_total"),
			"Class queue length",
			classlabels, nil,
		),
		requeues: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "requeque_total"),
			"Class requeque counter",
			classlabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (cc *ClassCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		cc.backlog,
		cc.bps,
		cc.bytes,
		cc.packets,
		cc.drops,
		cc.overlimits,
		cc.pps,
		cc.qlen,
		cc.requeues,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (cc *ClassCollector) Collect(ch chan<- prometheus.Metric) {
	// First we go and get the hostname of the system, so it can later be used in the labels
	host, err := os.Hostname()
	if err != nil {
		cc.logger.Info("failed to get hostname", "err", err)
	}

	// start iterating over the defined namespaces and devices
	for ns, devices := range cc.netns {
		// interate over each device, TODO: maybe there is a more elegant way to do this then 2 for
		// loops, I need a Go wizard to have a look at this.
		for _, interf := range devices {
			// Get all TC classes  for the specified device
			classes, err := getClasses(uint32(interf.Index), ns)
			if err != nil {
				cc.logger.Error("failed to get classes", "interface", interf.Attributes.Name, "err", err)
			}

			// Range over each class and report the statisctics of the class to the channel for Prometheus
			// metrics. Note that we print the handle with %x, so the hexadecimal notation. This way the
			// reported labels match the output from `tc -s show class ...`
			for _, cl := range classes {
				handleMaj, handleMin := HandleStr(cl.Handle)
				parentMaj, parentMin := HandleStr(cl.Parent)

				ch <- prometheus.MustNewConstMetric(
					cc.bytes,
					prometheus.CounterValue,
					float64(cl.Stats.Bytes),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.packets,
					prometheus.CounterValue,
					float64(cl.Stats.Packets),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.bps,
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
					cc.pps,
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
				ch <- prometheus.MustNewConstMetric(
					cc.backlog,
					prometheus.CounterValue,
					float64(cl.Stats.Backlog),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.drops,
					prometheus.CounterValue,
					float64(cl.Stats.Drops),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.overlimits,
					prometheus.CounterValue,
					float64(cl.Stats.Overlimits),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.qlen,
					prometheus.CounterValue,
					float64(cl.Stats.Qlen),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.requeues,
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
			}

		}
	}

}

