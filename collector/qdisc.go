package collector

import (
	"os"
	"strconv"

	netlink "github.com/fbegyn/netlink-vishv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	qdisclabels []string = []string{"host", "linkindex", "link", "type", "handle", "parent"}
)

// Collector for a generic Qdisc
type genericQdiscCollector struct {
	link   string
	qdisc  *netlink.GenericQdisc
	refcnt *prometheus.Desc
}

func NewGenericQdiscCollector(qdisc netlink.Qdisc, link string) (Collector, error) {
	return &genericQdiscCollector{
		link:  link,
		qdisc: qdisc.(*netlink.GenericQdisc),
		refcnt: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "qdisc", "refcnt"),
			"Qdisc refcnt",
			qdisclabels, nil,
		),
	}, nil
}

func (c *genericQdiscCollector) Update(ch chan<- prometheus.Metric) error {
	host, err := os.Hostname()
	if err != nil {
		logrus.Errorf("couldn't get host name: %v\n", err)
		return err
	}
	linkindex := strconv.Itoa(c.qdisc.Attrs().LinkIndex)

	ch <- prometheus.MustNewConstMetric(
		c.refcnt,
		prometheus.GaugeValue,
		float64(c.qdisc.Attrs().Refcnt),
		host,
		linkindex,
		c.link,
		c.qdisc.Type(),
		netlink.HandleStr(c.qdisc.Attrs().Handle),
		netlink.HandleStr(c.qdisc.Attrs().Parent),
	)
	return nil
}

// collector for a HFSC qdisc
type hfscQdiscCollector struct {
	link   string
	qdisc  *netlink.Hfsc
	refcnt *prometheus.Desc
	def    *prometheus.Desc
}

func NewHfscQdiscCollector(qdisc netlink.Qdisc, link string) (Collector, error) {
	module := "hfsc"
	return &hfscQdiscCollector{
		link:  link,
		qdisc: qdisc.(*netlink.Hfsc),
		refcnt: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "qdisc", "refcnt"),
			"Qdisc refcnt",
			qdisclabels, nil,
		),
		def: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, module, "default"),
			"Default class id for the HFSC qdisc",
			qdisclabels, nil,
		),
	}, nil
}

func (c *hfscQdiscCollector) Update(ch chan<- prometheus.Metric) error {
	host, err := os.Hostname()
	if err != nil {
		logrus.Errorf("couldn't get host name: %v\n", err)
		return err
	}
	linkindex := strconv.Itoa(c.qdisc.Attrs().LinkIndex)

	ch <- prometheus.MustNewConstMetric(
		c.refcnt,
		prometheus.GaugeValue,
		float64(c.qdisc.Attrs().Refcnt),
		host,
		linkindex,
		c.link,
		c.qdisc.Type(),
		netlink.HandleStr(c.qdisc.Attrs().Handle),
		netlink.HandleStr(c.qdisc.Attrs().Parent),
	)
	ch <- prometheus.MustNewConstMetric(
		c.refcnt,
		prometheus.GaugeValue,
		float64(c.qdisc.Defcls),
		host,
		linkindex,
		c.link,
		c.qdisc.Type(),
		netlink.HandleStr(c.qdisc.Attrs().Handle),
		netlink.HandleStr(c.qdisc.Attrs().Parent),
	)
	return nil
}

// collector for fq_codel qdisc
type fqcodelCollector struct {
	link     string
	qdisc    *netlink.FqCodel
	refcnt   *prometheus.Desc
	target   *prometheus.Desc
	limit    *prometheus.Desc
	interval *prometheus.Desc
	ecn      *prometheus.Desc
	flows    *prometheus.Desc
	quantum  *prometheus.Desc
}

func NewFqcodelQdiscCollector(qdisc netlink.Qdisc, link string) (Collector, error) {
	module := "fqcodel"
	return &fqcodelCollector{
		link:  link,
		qdisc: qdisc.(*netlink.FqCodel),
		refcnt: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "qdisc", "refcnt"),
			"Qdisc refcnt",
			qdisclabels, nil,
		),
		target: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, module, "target"),
			"The acceptable minimum standing/persistent queue delay",
			qdisclabels, nil,
		),
		limit: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, module, "limit"),
			"The hard limit on the real queue size",
			qdisclabels, nil,
		),
		interval: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, module, "interval"),
			"Used to ensure that the measured minimum delay does not become too stale",
			qdisclabels, nil,
		),
		ecn: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, module, "ecn"),
			"Can be used to mark packets instead of dropping them",
			qdisclabels, nil,
		),
		flows: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, module, "flows"),
			"The number of flows into which the incoming packets are classified",
			qdisclabels, nil,
		),
		quantum: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, module, "quantum"),
			"The number of flows into which the incoming packets are classified",
			qdisclabels, nil,
		),
	}, nil
}

func (c *fqcodelCollector) Update(ch chan<- prometheus.Metric) error {
	host, err := os.Hostname()
	if err != nil {
		logrus.Errorf("couldn't get host name: %v\n", err)
		return err
	}
	linkindex := strconv.Itoa(c.qdisc.Attrs().LinkIndex)

	ch <- prometheus.MustNewConstMetric(
		c.refcnt,
		prometheus.GaugeValue,
		float64(c.qdisc.Attrs().Refcnt),
		host,
		linkindex,
		c.link,
		c.qdisc.Type(),
		netlink.HandleStr(c.qdisc.Attrs().Handle),
		netlink.HandleStr(c.qdisc.Attrs().Parent),
	)
	ch <- prometheus.MustNewConstMetric(
		c.target,
		prometheus.GaugeValue,
		float64(c.qdisc.Target),
		host,
		linkindex,
		c.link,
		c.qdisc.Type(),
		netlink.HandleStr(c.qdisc.Attrs().Handle),
		netlink.HandleStr(c.qdisc.Attrs().Parent),
	)
	ch <- prometheus.MustNewConstMetric(
		c.limit,
		prometheus.GaugeValue,
		float64(c.qdisc.Limit),
		host,
		linkindex,
		c.link,
		c.qdisc.Type(),
		netlink.HandleStr(c.qdisc.Attrs().Handle),
		netlink.HandleStr(c.qdisc.Attrs().Parent),
	)
	ch <- prometheus.MustNewConstMetric(
		c.interval,
		prometheus.GaugeValue,
		float64(c.qdisc.Interval),
		host,
		linkindex,
		c.link,
		c.qdisc.Type(),
		netlink.HandleStr(c.qdisc.Attrs().Handle),
		netlink.HandleStr(c.qdisc.Attrs().Parent),
	)
	ch <- prometheus.MustNewConstMetric(
		c.ecn,
		prometheus.GaugeValue,
		float64(c.qdisc.ECN),
		host,
		linkindex,
		c.link,
		c.qdisc.Type(),
		netlink.HandleStr(c.qdisc.Attrs().Handle),
		netlink.HandleStr(c.qdisc.Attrs().Parent),
	)
	ch <- prometheus.MustNewConstMetric(
		c.flows,
		prometheus.GaugeValue,
		float64(c.qdisc.Flows),
		host,
		linkindex,
		c.link,
		c.qdisc.Type(),
		netlink.HandleStr(c.qdisc.Attrs().Handle),
		netlink.HandleStr(c.qdisc.Attrs().Parent),
	)
	ch <- prometheus.MustNewConstMetric(
		c.quantum,
		prometheus.GaugeValue,
		float64(c.qdisc.Quantum),
		host,
		linkindex,
		c.link,
		c.qdisc.Type(),
		netlink.HandleStr(c.qdisc.Attrs().Handle),
		netlink.HandleStr(c.qdisc.Attrs().Parent),
	)
	return nil
}
