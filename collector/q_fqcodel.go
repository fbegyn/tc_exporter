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
	fqCodelLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

// FqCodelQdiscCollector is the object that will collect fq_codel qdisc data for the interface
type FqCodelQdiscCollector struct {
	logger         slog.Logger
	netns          map[string][]rtnetlink.LinkMessage
	ceMark         *prometheus.Desc
	dropOverlimit  *prometheus.Desc
	dropOvermemory *prometheus.Desc
	ecnMark        *prometheus.Desc
	maxPacket      *prometheus.Desc
	memoryUsage    *prometheus.Desc
	newFlowCount   *prometheus.Desc
	newFlowsLen    *prometheus.Desc
	oldFlowsLen    *prometheus.Desc
}

// NewFqCodelQdiscCollector create a new QdiscCollector given a network interface
func NewFqCodelQdiscCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (TcSubCollector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "fq_codel")
	log.Debug("making fq_codel qdisc collector")

	return &FqCodelQdiscCollector{
		logger: *log,
		netns:  netns,
		ceMark: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fqcodel", "ce_mark"),
			"fq_codel packets above ce-threshold",
			fqCodelLabels, nil,
		),
		dropOverlimit: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fqcodel", "drop_overlimit"),
			"fq_codel number of times max qdisc packet limit was hit",
			fqCodelLabels, nil,
		),
		dropOvermemory: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fqcodel", "drop_overmemory"),
			"fq_codel drop overmemory xstat",
			fqCodelLabels, nil,
		),
		ecnMark: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fqcodel", "ecn_mark"),
			"fq_codel nmber of packets we ECN marked instead of being dropped",
			fqCodelLabels, nil,
		),
		maxPacket: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fqcodel", "max_packet"),
			"fq_codel largest packet we’ve seen so far",
			fqCodelLabels, nil,
		),
		memoryUsage: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fqcodel", "memory_usage"),
			"fq_codel memory usage in bytes",
			fqCodelLabels, nil,
		),
		newFlowCount: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fqcodel", "new_flows_count"),
			"fq_codel number of times packets created a new flow",
			fqCodelLabels, nil,
		),
		newFlowsLen: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fqcodel", "new_flows_len"),
			"fq_codel count of flows in new list",
			fqCodelLabels, nil,
		),
		oldFlowsLen: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "fqcodel", "old_flows_len"),
			"fq_codel count of flows in old list",
			fqCodelLabels, nil,
		),
	}, nil
}

// Describe implements Collector
func (col *FqCodelQdiscCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		col.ceMark,
		col.dropOverlimit,
		col.dropOvermemory,
		col.ecnMark,
		col.maxPacket,
		col.memoryUsage,
		col.newFlowCount,
		col.newFlowsLen,
		col.oldFlowsLen,
	}

	for _, d := range ds {
		ch <- d
	}
}

// Collect fetches and updates the data the collector is exporting
func (col *FqCodelQdiscCollector) Collect(ch chan<- prometheus.Metric, objects map[string]map[string][]tc.Object) {
	// fetch the host for useage later on
	host, err := os.Hostname()
	if err != nil {
		col.logger.Error("failed to fetch hostname", "err", err)
	}

	// iterate through the netns and devices
	for ns, devices := range col.netns {
		for _, interf := range devices {
			// fetch all the the qdisc for this interface
			// qdiscs, err := getQdiscs(uint32(interf.Index), ns)
			// if err != nil {
			// 	col.logger.Error("failed to get qdiscs", "interface", interf.Attributes.Name, "err", err)
			// }

			// iterate through all the qdiscs and sent the data to the prometheus metric channel
			for _, qd := range objects[ns]["qdisc"] {
				if qd.FqCodel == nil || qd.Msg.Ifindex != interf.Index {
					continue
				}
				handleMaj, handleMin := HandleStr(qd.Handle)
				parentMaj, parentMin := HandleStr(qd.Parent)

				ch <- prometheus.MustNewConstMetric(
					col.ceMark,
					prometheus.CounterValue,
					float64(qd.XStats.FqCodel.Qd.CeMark),
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
					float64(qd.XStats.FqCodel.Qd.DropOverlimit),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.dropOvermemory,
					prometheus.CounterValue,
					float64(qd.XStats.FqCodel.Qd.DropOvermemory),
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
					float64(qd.XStats.FqCodel.Qd.EcnMark),
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
					float64(qd.XStats.FqCodel.Qd.MaxPacket),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.memoryUsage,
					prometheus.CounterValue,
					float64(qd.XStats.FqCodel.Qd.MemoryUsage),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.newFlowCount,
					prometheus.CounterValue,
					float64(qd.XStats.FqCodel.Qd.NewFlowCount),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.newFlowsLen,
					prometheus.CounterValue,
					float64(qd.XStats.FqCodel.Qd.NewFlowsLen),
					host,
					ns,
					fmt.Sprintf("%d", interf.Index),
					interf.Attributes.Name,
					qd.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					col.oldFlowsLen,
					prometheus.CounterValue,
					float64(qd.XStats.FqCodel.Qd.OldFlowsLen),
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
