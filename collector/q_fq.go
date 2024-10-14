package tccollector

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	fqLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// FqCollector is the object that will collect FQ qdisc data for the interface
type FqCollector struct {
	logger slog.Logger
	netns  map[string][]rtnetlink.LinkMessage

	gcFlows             *prometheus.Desc // uint64
	highPrioPackets     *prometheus.Desc // uint64
	tcpRetrans          *prometheus.Desc // uint64
	throttled           *prometheus.Desc // uint64
	flowsPlimit         *prometheus.Desc // uint64
	pktsTooLong         *prometheus.Desc // uint64
	allocationErrors    *prometheus.Desc // uint64
	timeNextDelayedFlow *prometheus.Desc // int64
	flows               *prometheus.Desc // uint32
	inactiveFlows       *prometheus.Desc // uint32
	throttledFlows      *prometheus.Desc // uint32
	unthrottleLatencyNs *prometheus.Desc // uint32
	ceMark              *prometheus.Desc // uint64
	horizonDrops        *prometheus.Desc // uint64
	horizonCaps         *prometheus.Desc // uint64
	fastpathPackets     *prometheus.Desc // uint64
}

// NewFqCollector create a new QdiscCollector given a network interface
func NewFqCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (prometheus.Collector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "fq")
	log.Info("making fq collector")

	return &FqCollector{
		logger: *log,
		netns:  netns,
		gcFlows: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "gc_flows"),
			"FQ gc flow counter",
			fqLabels, nil,
		),
		highPrioPackets: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "high_prio_packets"),
			"FQ high prio packets",
			fqLabels, nil,
		),
		tcpRetrans: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "tcp_retrans"),
			"FQ TCP retransmits",
			fqLabels, nil,
		),
		throttled: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "throttled"),
			"FQ throttled",
			fqLabels, nil,
		),
		throttledFlows: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "throttled_flows"),
			"FQ throttled flows",
			fqLabels, nil,
		),
		flowsPlimit: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "flows_p_limit"),
			"FQ flows p limt",
			fqLabels, nil,
		),
		pktsTooLong: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "packets_too_long"),
			"FQ packets too long",
			fqLabels, nil,
		),
		allocationErrors: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "allocation_errors"),
			"FQ allocation errors",
			fqLabels, nil,
		),
		timeNextDelayedFlow: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "time_next_delayed_flow"),
			"FQ time nexted delayed flow",
			fqLabels, nil,
		),
		flows: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "flows"),
			"FQ flows",
			fqLabels, nil,
		),
		inactiveFlows: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "inactive_flows"),
			"FQ inactive flows",
			fqLabels, nil,
		),
		unthrottleLatencyNs: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "unthrottled_latency_ns"),
			"FQ unthrottled latency in ns",
			fqLabels, nil,
		),
		ceMark: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "ce_mark"),
			"FQ ce mark",
			fqLabels, nil,
		),
		horizonDrops: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "horizon_drops"),
			"FQ horizon drops",
			fqLabels, nil,
		),
		horizonCaps: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "horizon_caps"),
			"FQ horizon caps",
			fqLabels, nil,
		),
		fastpathPackets: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fq", "fast_path_packets"),
			"FQ fast path packets",
			fqLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (col *FqCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		col.allocationErrors,
		col.ceMark,
		col.fastpathPackets,
		col.flows,
		col.flowsPlimit,
		col.gcFlows,
		col.highPrioPackets,
		col.horizonCaps,
		col.horizonDrops,
		col.inactiveFlows,
		col.pktsTooLong,
		col.tcpRetrans,
		col.throttled,
		col.throttledFlows,
		col.timeNextDelayedFlow,
		col.unthrottleLatencyNs,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (col *FqCollector) Collect(ch chan<- prometheus.Metric) {
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
				if qd.Fq == nil {
					continue
				}
				handleMaj, handleMin := HandleStr(qd.Handle)
				parentMaj, parentMin := HandleStr(qd.Parent)

				ch <- prometheus.MustNewConstMetric(
					col.gcFlows,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.GcFlows),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.highPrioPackets,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.HighPrioPackets),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.tcpRetrans,
					prometheus.GaugeValue,
					float64(qd.XStats.Fq.TCPRetrans),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.throttled,
					prometheus.GaugeValue,
					float64(qd.XStats.Fq.Throttled),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.flowsPlimit,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.FlowsPlimit),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.pktsTooLong,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.PktsTooLong),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.allocationErrors,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.AllocationErrors),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.timeNextDelayedFlow,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.TimeNextDelayedFlow),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.flows,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.Flows),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.inactiveFlows,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.InactiveFlows),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.throttledFlows,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.ThrottledFlows),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.ceMark,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.CEMark),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.horizonDrops,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.HorizonDrops),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.horizonCaps,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.HorizonCaps),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.fastpathPackets,
					prometheus.CounterValue,
					float64(qd.XStats.Fq.FastpathPackets),
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
