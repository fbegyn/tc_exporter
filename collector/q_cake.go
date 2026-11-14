package tccollector

import (
	"fmt"
	"log/slog"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	cakeLabels    []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
	cakeTinLabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent", "tin"}
)

// CakeCollector is the object that will collect cake qdisc data for the interface
type CakeCollector struct {
	logger           slog.Logger
	netns            map[string][]rtnetlink.LinkMessage
	capacityEstimate *prometheus.Desc
	memoryLimit      *prometheus.Desc
	memoryUsed       *prometheus.Desc
	avgNetoff        *prometheus.Desc
	minNetlen        *prometheus.Desc
	maxNetlen        *prometheus.Desc
	minAdjlen        *prometheus.Desc
	maxAdjlen        *prometheus.Desc
	activeQueues     *prometheus.Desc
	tin              cakeTinDescs
}

// cakeTinDescs holds the descriptions of the per tin statistics
type cakeTinDescs struct {
	sentPackets        *prometheus.Desc
	sentBytes          *prometheus.Desc
	droppedPackets     *prometheus.Desc
	droppedBytes       *prometheus.Desc
	acksDroppedPackets *prometheus.Desc
	acksDroppedBytes   *prometheus.Desc
	ecnMarkedPackets   *prometheus.Desc
	ecnMarkedBytes     *prometheus.Desc
	backlogPackets     *prometheus.Desc
	backlogBytes       *prometheus.Desc
	thresholdRate      *prometheus.Desc
	targetUs           *prometheus.Desc
	intervalUs         *prometheus.Desc
	wayIndirectHits    *prometheus.Desc
	wayMisses          *prometheus.Desc
	wayCollisions      *prometheus.Desc
	peakDelayUs        *prometheus.Desc
	avgDelayUs         *prometheus.Desc
	baseDelayUs        *prometheus.Desc
	sparseFlows        *prometheus.Desc
	bulkFlows          *prometheus.Desc
	unresponsiveFlows  *prometheus.Desc
	maxSkblen          *prometheus.Desc
	flowQuantum        *prometheus.Desc
}

func init() {
	registerCollector("cake", false, NewCakeCollector)
}

// NewCakeCollector create a new QdiscCollector given a network interface
func NewCakeCollector(netns map[string][]rtnetlink.LinkMessage, log *slog.Logger) (ObjectCollector, error) {
	// Setup logger for qdisc collector
	log = log.With("collector", "cake")
	log.Info("making cake qdisc collector")

	desc := func(name, help string) *prometheus.Desc {
		return prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cake", name),
			help,
			cakeLabels, nil,
		)
	}
	tinDesc := func(name, help string) *prometheus.Desc {
		return prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "cake", name),
			help,
			cakeTinLabels, nil,
		)
	}

	return &CakeCollector{
		logger:           *log,
		netns:            netns,
		capacityEstimate: desc("capacity_estimate", "cake capacity estimate xstat in bytes per second"),
		memoryLimit:      desc("memory_limit", "cake memory limit xstat"),
		memoryUsed:       desc("memory_used", "cake memory used xstat"),
		avgNetoff:        desc("avg_netoff", "cake average network offset xstat"),
		minNetlen:        desc("min_netlen", "cake minimum network length xstat"),
		maxNetlen:        desc("max_netlen", "cake maximum network length xstat"),
		minAdjlen:        desc("min_adjlen", "cake minimum adjusted length xstat"),
		maxAdjlen:        desc("max_adjlen", "cake maximum adjusted length xstat"),
		activeQueues:     desc("active_queues", "cake active queues xstat"),
		tin: cakeTinDescs{
			sentPackets:        tinDesc("tin_sent_packets", "cake tin sent packets xstat"),
			sentBytes:          tinDesc("tin_sent_bytes", "cake tin sent bytes xstat"),
			droppedPackets:     tinDesc("tin_dropped_packets", "cake tin dropped packets xstat"),
			droppedBytes:       tinDesc("tin_dropped_bytes", "cake tin dropped bytes xstat"),
			acksDroppedPackets: tinDesc("tin_acks_dropped_packets", "cake tin acks dropped packets xstat"),
			acksDroppedBytes:   tinDesc("tin_acks_dropped_bytes", "cake tin acks dropped bytes xstat"),
			ecnMarkedPackets:   tinDesc("tin_ecn_marked_packets", "cake tin ecn marked packets xstat"),
			ecnMarkedBytes:     tinDesc("tin_ecn_marked_bytes", "cake tin ecn marked bytes xstat"),
			backlogPackets:     tinDesc("tin_backlog_packets", "cake tin backlog packets xstat"),
			backlogBytes:       tinDesc("tin_backlog_bytes", "cake tin backlog bytes xstat"),
			thresholdRate:      tinDesc("tin_threshold_rate", "cake tin threshold rate xstat in bytes per second"),
			targetUs:           tinDesc("tin_target_us", "cake tin target xstat in microseconds"),
			intervalUs:         tinDesc("tin_interval_us", "cake tin interval xstat in microseconds"),
			wayIndirectHits:    tinDesc("tin_way_indirect_hits", "cake tin way indirect hits xstat"),
			wayMisses:          tinDesc("tin_way_misses", "cake tin way misses xstat"),
			wayCollisions:      tinDesc("tin_way_collisions", "cake tin way collisions xstat"),
			peakDelayUs:        tinDesc("tin_peak_delay_us", "cake tin peak delay xstat in microseconds"),
			avgDelayUs:         tinDesc("tin_avg_delay_us", "cake tin average delay xstat in microseconds"),
			baseDelayUs:        tinDesc("tin_base_delay_us", "cake tin base delay xstat in microseconds"),
			sparseFlows:        tinDesc("tin_sparse_flows", "cake tin sparse flows xstat"),
			bulkFlows:          tinDesc("tin_bulk_flows", "cake tin bulk flows xstat"),
			unresponsiveFlows:  tinDesc("tin_unresponsive_flows", "cake tin unresponsive flows xstat"),
			maxSkblen:          tinDesc("tin_max_skblen", "cake tin maximum skb length xstat"),
			flowQuantum:        tinDesc("tin_flow_quantum", "cake tin flow quantum xstat"),
		},
	}, nil
}

// Describe implements Collector
func (col *CakeCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		col.capacityEstimate,
		col.memoryLimit,
		col.memoryUsed,
		col.avgNetoff,
		col.minNetlen,
		col.maxNetlen,
		col.minAdjlen,
		col.maxAdjlen,
		col.activeQueues,
		col.tin.sentPackets,
		col.tin.sentBytes,
		col.tin.droppedPackets,
		col.tin.droppedBytes,
		col.tin.acksDroppedPackets,
		col.tin.acksDroppedBytes,
		col.tin.ecnMarkedPackets,
		col.tin.ecnMarkedBytes,
		col.tin.backlogPackets,
		col.tin.backlogBytes,
		col.tin.thresholdRate,
		col.tin.targetUs,
		col.tin.intervalUs,
		col.tin.wayIndirectHits,
		col.tin.wayMisses,
		col.tin.wayCollisions,
		col.tin.peakDelayUs,
		col.tin.avgDelayUs,
		col.tin.baseDelayUs,
		col.tin.sparseFlows,
		col.tin.bulkFlows,
		col.tin.unresponsiveFlows,
		col.tin.maxSkblen,
		col.tin.flowQuantum,
	}

	for _, d := range ds {
		ch <- d
	}
}

// CollectObject fetches and updates the data the collector is exporting
func (col *CakeCollector) CollectObject(ch chan<- prometheus.Metric, host, ns string, interf rtnetlink.LinkMessage, qd tc.Object) {
	xs := qd.XStats.Cake
	if xs == nil {
		col.logger.Debug("cake xstats are empty for this qdisc", "qdisc", qd)
		return
	}

	handleMaj, handleMin := HandleStr(qd.Handle)
	parentMaj, parentMin := HandleStr(qd.Parent)
	labels := []string{
		host,
		ns,
		fmt.Sprintf("%d", interf.Index),
		interf.Attributes.Name,
		qd.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	}

	emitCakeStat(ch, col.capacityEstimate, prometheus.GaugeValue, xs.CapacityEstimate, labels)
	emitCakeStat(ch, col.memoryLimit, prometheus.GaugeValue, xs.MemoryLimit, labels)
	emitCakeStat(ch, col.memoryUsed, prometheus.GaugeValue, xs.MemoryUsed, labels)
	emitCakeStat(ch, col.avgNetoff, prometheus.GaugeValue, xs.AvgNetoff, labels)
	emitCakeStat(ch, col.minNetlen, prometheus.GaugeValue, xs.MinNetlen, labels)
	emitCakeStat(ch, col.maxNetlen, prometheus.GaugeValue, xs.MaxNetlen, labels)
	emitCakeStat(ch, col.minAdjlen, prometheus.GaugeValue, xs.MinAdjlen, labels)
	emitCakeStat(ch, col.maxAdjlen, prometheus.GaugeValue, xs.MaxAdjlen, labels)
	emitCakeStat(ch, col.activeQueues, prometheus.GaugeValue, xs.ActiveQueues, labels)

	for i, tin := range xs.Tins {
		tinLabels := append(labels[:len(labels):len(labels)], fmt.Sprintf("%d", i))

		emitCakeStat(ch, col.tin.sentPackets, prometheus.CounterValue, tin.SentPackets, tinLabels)
		emitCakeStat(ch, col.tin.sentBytes, prometheus.CounterValue, tin.SentBytes, tinLabels)
		emitCakeStat(ch, col.tin.droppedPackets, prometheus.CounterValue, tin.DroppedPackets, tinLabels)
		emitCakeStat(ch, col.tin.droppedBytes, prometheus.CounterValue, tin.DroppedBytes, tinLabels)
		emitCakeStat(ch, col.tin.acksDroppedPackets, prometheus.CounterValue, tin.AcksDroppedPackets, tinLabels)
		emitCakeStat(ch, col.tin.acksDroppedBytes, prometheus.CounterValue, tin.AcksDroppedBytes, tinLabels)
		emitCakeStat(ch, col.tin.ecnMarkedPackets, prometheus.CounterValue, tin.EcnMarkedPackets, tinLabels)
		emitCakeStat(ch, col.tin.ecnMarkedBytes, prometheus.CounterValue, tin.EcnMarkedBytes, tinLabels)
		emitCakeStat(ch, col.tin.backlogPackets, prometheus.GaugeValue, tin.BacklogPackets, tinLabels)
		emitCakeStat(ch, col.tin.backlogBytes, prometheus.GaugeValue, tin.BacklogBytes, tinLabels)
		emitCakeStat(ch, col.tin.thresholdRate, prometheus.GaugeValue, tin.ThresholdRate, tinLabels)
		emitCakeStat(ch, col.tin.targetUs, prometheus.GaugeValue, tin.TargetUs, tinLabels)
		emitCakeStat(ch, col.tin.intervalUs, prometheus.GaugeValue, tin.IntervalUs, tinLabels)
		emitCakeStat(ch, col.tin.wayIndirectHits, prometheus.CounterValue, tin.WayIndirectHits, tinLabels)
		emitCakeStat(ch, col.tin.wayMisses, prometheus.CounterValue, tin.WayMisses, tinLabels)
		emitCakeStat(ch, col.tin.wayCollisions, prometheus.CounterValue, tin.WayCollisions, tinLabels)
		emitCakeStat(ch, col.tin.peakDelayUs, prometheus.GaugeValue, tin.PeakDelayUs, tinLabels)
		emitCakeStat(ch, col.tin.avgDelayUs, prometheus.GaugeValue, tin.AvgDelayUs, tinLabels)
		emitCakeStat(ch, col.tin.baseDelayUs, prometheus.GaugeValue, tin.BaseDelayUs, tinLabels)
		emitCakeStat(ch, col.tin.sparseFlows, prometheus.GaugeValue, tin.SparseFlows, tinLabels)
		emitCakeStat(ch, col.tin.bulkFlows, prometheus.GaugeValue, tin.BulkFlows, tinLabels)
		emitCakeStat(ch, col.tin.unresponsiveFlows, prometheus.GaugeValue, tin.UnresponsiveFlows, tinLabels)
		emitCakeStat(ch, col.tin.maxSkblen, prometheus.GaugeValue, tin.MaxSkblen, tinLabels)
		emitCakeStat(ch, col.tin.flowQuantum, prometheus.GaugeValue, tin.FlowQuantum, tinLabels)
	}
}

// emitCakeStat sends the metric if the kernel reported the statistic, cake
// statistics are optional attributes and nil when not present
func emitCakeStat[T uint32 | uint64](ch chan<- prometheus.Metric, desc *prometheus.Desc, vt prometheus.ValueType, v *T, labels []string) {
	if v == nil {
		return
	}
	ch <- prometheus.MustNewConstMetric(desc, vt, float64(*v), labels...)
}
