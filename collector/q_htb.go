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
	htbLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// HtbCollector is the object that will collect htb qdisc data for the interface
type HtbCollector struct {
	logger  slog.Logger
	netns   map[string][]rtnetlink.LinkMessage
	borrows *prometheus.Desc
	cTokens *prometheus.Desc
	giants  *prometheus.Desc
	lends   *prometheus.Desc
	tokens  *prometheus.Desc
}

// NewHtbCollector create a new QdiscCollector given a network interface
func NewHtbCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (ObjectCollector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "htb")
	log.Info("making htb collector")

	return &HtbCollector{
		logger: *log,
		netns:  netns,
		borrows: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "htb", "borrows"),
			"HTB borrows xstat",
			htbLabels, nil,
		),
		cTokens: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "htb", "ctokens"),
			"HTB ctokens xstat",
			htbLabels, nil,
		),
		giants: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "htb", "giants"),
			"HTB giants xstat",
			htbLabels, nil,
		),
		lends: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "htb", "lends"),
			"HTB lends xstat",
			htbLabels, nil,
		),
		tokens: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "htb", "tokens"),
			"HTB tokens xstat",
			htbLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (col *HtbCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		col.borrows,
		col.cTokens,
		col.giants,
		col.lends,
		col.tokens,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (col *HtbCollector) Collect(ch chan<- prometheus.Metric) {
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
				if qd.Htb == nil || qd.XStats == nil {
					continue
				}
				handleMaj, handleMin := HandleStr(qd.Handle)
				parentMaj, parentMin := HandleStr(qd.Parent)

				ch <- prometheus.MustNewConstMetric(
					col.borrows,
					prometheus.CounterValue,
					float64(qd.XStats.Htb.Borrows),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.cTokens,
					prometheus.CounterValue,
					float64(qd.XStats.Htb.CTokens),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.giants,
					prometheus.CounterValue,
					float64(qd.XStats.Htb.Giants),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.lends,
					prometheus.CounterValue,
					float64(qd.XStats.Htb.Lends),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.tokens,
					prometheus.CounterValue,
					float64(qd.XStats.Htb.Tokens),
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
func (col *HtbCollector) CollectObject(ch chan<- prometheus.Metric, host, ns string, interf rtnetlink.LinkMessage, qd tc.Object) {
	if qd.XStats == nil {
		return
	}

	handleMaj, handleMin := HandleStr(qd.Handle)
	parentMaj, parentMin := HandleStr(qd.Parent)

	ch <- prometheus.MustNewConstMetric(
		col.borrows,
		prometheus.CounterValue,
		float64(qd.XStats.Htb.Borrows),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.cTokens,
		prometheus.CounterValue,
		float64(qd.XStats.Htb.CTokens),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.giants,
		prometheus.CounterValue,
		float64(qd.XStats.Htb.Giants),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.lends,
		prometheus.CounterValue,
		float64(qd.XStats.Htb.Lends),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		col.tokens,
		prometheus.CounterValue,
		float64(qd.XStats.Htb.Tokens),
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
}
