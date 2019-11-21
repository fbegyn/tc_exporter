package main

import (
	"net/http"
	"os"
	"strconv"

	netlink "github.com/fbegyn/netlink-vishv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

var (
	operstatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_link_operstatus",
			Help: "Operational status of the link from IFLA_OPERSTATE numeric representation of RFC2863",
		},
		[]string{"host", "name", "type", "hwaddr"},
	)
	qdiscRefcnt = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_qdisc_refcnt",
			Help: "Qdisc refcount",
		},
		[]string{"host", "linkindex", "type", "handle", "parent"},
	)
	hfscDefault = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_qdisc_hfsc_default",
			Help: "Default class id for the HFSC qdisc",
		},
		[]string{"host", "linkindex", "type", "handle", "parent"},
	)
	fqcodelTarget = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_qdisc_fqcodel_target",
			Help: "The acceptable minimum standing/persistent queue delay",
		},
		[]string{"host", "linkindex", "type", "handle", "parent"},
	)
	fqcodelLimit = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_qdisc_fqcodel_limit",
			Help: "The hard limit on the real queue size",
		},
		[]string{"host", "linkindex", "type", "handle", "parent"},
	)
	fqcodelInterval = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_qdisc_fqcodel_interval",
			Help: "Used to ensure that the measured minimum delay does not become too stale",
		},
		[]string{"host", "linkindex", "type", "handle", "parent"},
	)
	fqcodelECN = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_qdisc_fqcodel_ecn",
			Help: "Can be used to mark packets instead of dropping them",
		},
		[]string{"host", "linkindex", "type", "handle", "parent"},
	)
	fqcodelFlows = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_qdisc_fqcodel_flows",
			Help: "The number of flows into which the incoming packets are classified",
		},
		[]string{"host", "linkindex", "type", "handle", "parent"},
	)
	fqcodelQuantum = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_qdisc_fqcodel_quantum",
			Help: "The number of bytes used as 'deficit' in the fair queuing algorithm",
		},
		[]string{"host", "linkindex", "type", "handle", "parent"},
	)
	classBytes = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_class_bytes",
			Help: "Sent bytes",
		},
		[]string{"host", "linkindex", "type", "handle", "parent", "leaf"},
	)
	classPackets = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_class_packets",
			Help: "Sent packets",
		},
		[]string{"host", "linkindex", "type", "handle", "parent", "leaf"},
	)
	classBacklog = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_class_backlog",
			Help: "Packets in backlog",
		},
		[]string{"host", "linkindex", "type", "handle", "parent", "leaf"},
	)
	classDrops = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_class_drops",
			Help: "Dropped packets",
		},
		[]string{"host", "linkindex", "type", "handle", "parent", "leaf"},
	)
	classOverlimits = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_class_overlimits",
			Help: "Overlimit packets",
		},
		[]string{"host", "linkindex", "type", "handle", "parent", "leaf"},
	)
	classRequeues = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_class_requeues",
			Help: "Requeue packets",
		},
		[]string{"host", "linkindex", "type", "handle", "parent", "leaf"},
	)
	classQlen = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_class_qlen",
			Help: "Packets in qlen",
		},
		[]string{"host", "linkindex", "type", "handle", "parent", "leaf"},
	)
	classBps = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_class_bps",
			Help: "Bytes per second",
		},
		[]string{"host", "linkindex", "type", "handle", "parent", "leaf"},
	)
	classPps = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_class_pps",
			Help: "Packets per second",
		},
		[]string{"host", "linkindex", "type", "handle", "parent", "leaf"},
	)
	hfscSC = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "tc_class_hfsc_sc",
			Help: "Service curve for the hfsc class",
		},
		[]string{"host", "linkindex", "type", "handle", "parent", "leaf", "sc", "param"},
	)
)

// PromExporter starts the prometheus exporter listener on the provided port
func PromExporter() {
	http.Handle("/app", promhttp.Handler())
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Fetch the url encoded interface
		logrus.Infof("received request to scrape all interfaces\n")

		register := prometheus.NewRegistry()
		interfaces, err := netlink.LinkList()
		if err != nil {
			logrus.Fatalf("failed to scrape all interfaces\n")
		}
		// Create the register for the data
		exporter := NewTcExporer(interfaces)
		register.MustRegister(exporter)

		h := promhttp.HandlerFor(register, promhttp.HandlerOpts{})
		h.ServeHTTP(w, r)
	})
}

type TcExporter struct {
	interf []netlink.Link
}

func NewTcExporer(links []netlink.Link) *TcExporter {
	return &TcExporter{
		interf: links,
	}
}

func (c *TcExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- prometheus.NewDesc("dummy", "dummy", nil, nil)
}

func (c *TcExporter) Collect(ch chan<- prometheus.Metric) {
	// Fetch the required data for the exporter
	for _, link := range c.interf {
		data := GetData(link)
		collectLink(data.link)
		collectQdiscs(data.qdiscs)
		collectClasses(data.classes)
	}

	// Collect all the statistics
	// Register link stats
	operstatus.Collect(ch)
	// Register qdisc stats
	qdiscRefcnt.Collect(ch)
	hfscDefault.Collect(ch)
	fqcodelECN.Collect(ch)
	fqcodelFlows.Collect(ch)
	fqcodelInterval.Collect(ch)
	fqcodelLimit.Collect(ch)
	fqcodelQuantum.Collect(ch)
	fqcodelTarget.Collect(ch)
	// Regsiter class stats
	classBacklog.Collect(ch)
	classBps.Collect(ch)
	classBytes.Collect(ch)
	classDrops.Collect(ch)
	classOverlimits.Collect(ch)
	classPackets.Collect(ch)
	classPps.Collect(ch)
	classQlen.Collect(ch)
	classRequeues.Collect(ch)
	hfscSC.Collect(ch)
}

func collectLink(l netlink.Link) {
	host, err := os.Hostname()
	if err != nil {
		logrus.Errorf("couldn't get host name: %v\n", err)
	}

	// set operstatus according to https://tools.ietf.org/html/rfc2863#section-3.1.12
	operstatus.WithLabelValues(host, l.Attrs().Name, l.Type(), l.Attrs().HardwareAddr.String()).Set(float64(l.Attrs().OperState))
}

func collectQdiscs(qdiscs *[]netlink.Qdisc) {
	for _, q := range *qdiscs {
		collectQdisc(q)
	}
}

func collectQdisc(q netlink.Qdisc) {
	host, err := os.Hostname()
	if err != nil {
		logrus.Errorf("couldn't get host name: %v\n", err)
	}
	linkindex := strconv.Itoa(q.Attrs().LinkIndex)
	qdiscType := q.Type()
	handle := netlink.HandleStr(q.Attrs().Handle)
	parent := netlink.HandleStr(q.Attrs().Parent)
	qdiscRefcnt.WithLabelValues(host, linkindex, qdiscType, handle, parent).Set(float64(q.Attrs().Refcnt))
	switch qdiscType {
	case "hfsc":
		hfscQdisc := q.(*netlink.Hfsc)
		hfscDefault.WithLabelValues(host, linkindex, qdiscType, handle, parent).Set(float64(hfscQdisc.Defcls))
	case "fq_codel":
		fqcodelQdisc := q.(*netlink.FqCodel)
		fqcodelTarget.WithLabelValues(host, linkindex, qdiscType, handle, parent).Set(float64(fqcodelQdisc.Target + 1))
		fqcodelLimit.WithLabelValues(host, linkindex, qdiscType, handle, parent).Set(float64(fqcodelQdisc.Limit))
		fqcodelInterval.WithLabelValues(host, linkindex, qdiscType, handle, parent).Set(float64(fqcodelQdisc.Interval + 1))
		fqcodelECN.WithLabelValues(host, linkindex, qdiscType, handle, parent).Set(float64(fqcodelQdisc.ECN))
		fqcodelFlows.WithLabelValues(host, linkindex, qdiscType, handle, parent).Set(float64(fqcodelQdisc.Flows))
		fqcodelQuantum.WithLabelValues(host, linkindex, qdiscType, handle, parent).Set(float64(fqcodelQdisc.Quantum))
	}
}

func collectClasses(classes *[]netlink.Class) {
	for _, c := range *classes {
		collectClass(c)
	}
}

func collectClass(c netlink.Class) {
	host, err := os.Hostname()
	if err != nil {
		logrus.Errorf("couldn't get host name: %v\n", err)
	}
	linkindex := strconv.Itoa(c.Attrs().LinkIndex)
	classType := c.Type()
	handle := netlink.HandleStr(c.Attrs().Handle)
	parent := netlink.HandleStr(c.Attrs().Parent)
	leaf := netlink.HandleStr(c.Attrs().Leaf)
	classBytes.WithLabelValues(host, linkindex, classType, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Basic.Bytes))
	classPackets.WithLabelValues(host, linkindex, classType, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Basic.Packets))
	classBacklog.WithLabelValues(host, linkindex, classType, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Queue.Backlog))
	classDrops.WithLabelValues(host, linkindex, classType, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Queue.Drops))
	classOverlimits.WithLabelValues(host, linkindex, classType, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Queue.Overlimits))
	classRequeues.WithLabelValues(host, linkindex, classType, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Queue.Requeues))
	classQlen.WithLabelValues(host, linkindex, classType, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Queue.Qlen))
	classBps.WithLabelValues(host, linkindex, classType, handle, parent, leaf).Set(float64(c.Attrs().Statistics.RateEst.Bps))
	classPps.WithLabelValues(host, linkindex, classType, handle, parent, leaf).Set(float64(c.Attrs().Statistics.RateEst.Pps))
	switch classType {
	case "hfsc":
		hfscClass := c.(*netlink.HfscClass)
		Fburst, Fdelay, Frate := hfscClass.Fsc.Attrs()
		Uburst, Udelay, Urate := hfscClass.Usc.Attrs()
		Rburst, Rdelay, Rrate := hfscClass.Rsc.Attrs()
		hfscSC.WithLabelValues(host, linkindex, classType, handle, parent, leaf, "fsc", "burst").Set(float64(Fburst))
		hfscSC.WithLabelValues(host, linkindex, classType, handle, parent, leaf, "fsc", "delay").Set(float64(Fdelay))
		hfscSC.WithLabelValues(host, linkindex, classType, handle, parent, leaf, "fsc", "rate").Set(float64(Frate))
		hfscSC.WithLabelValues(host, linkindex, classType, handle, parent, leaf, "rsc", "burst").Set(float64(Rburst))
		hfscSC.WithLabelValues(host, linkindex, classType, handle, parent, leaf, "rsc", "delay").Set(float64(Rdelay))
		hfscSC.WithLabelValues(host, linkindex, classType, handle, parent, leaf, "rsc", "rate").Set(float64(Rrate))
		hfscSC.WithLabelValues(host, linkindex, classType, handle, parent, leaf, "usc", "burst").Set(float64(Uburst))
		hfscSC.WithLabelValues(host, linkindex, classType, handle, parent, leaf, "usc", "delay").Set(float64(Udelay))
		hfscSC.WithLabelValues(host, linkindex, classType, handle, parent, leaf, "usc", "rate").Set(float64(Urate))
	}
}
