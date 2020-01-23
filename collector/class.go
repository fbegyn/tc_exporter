package collector

import (
	"os"
	"strconv"

	netlink "github.com/fbegyn/netlink-vishv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	classlabels []string = []string{"host", "linkindex", "link", "type", "handle", "parent", "leaf"}
)

// Collector for a generic Class
type genericClassCollector struct {
	link       string
	class      *netlink.GenericClass
	bytes      *prometheus.Desc
	packets    *prometheus.Desc
	backlog    *prometheus.Desc
	drops      *prometheus.Desc
	overlimits *prometheus.Desc
	requeues   *prometheus.Desc
	qlen       *prometheus.Desc
	bps        *prometheus.Desc
	pps        *prometheus.Desc
}

func NewGenericClassCollector(class netlink.Class, link string) (Collector, error) {
	return &genericClassCollector{
		link:  link,
		class: class.(*netlink.GenericClass),
		bytes: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "bytes"),
			"Bytes passed though class",
			classlabels, nil,
		),
		packets: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "packets"),
			"Packets passed through class",
			classlabels, nil,
		),
		backlog: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "backlog"),
			"Class backlog",
			classlabels, nil,
		),
		drops: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "drops"),
			"Class drops",
			classlabels, nil,
		),
		overlimits: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "overlimits"),
			"Class overlimits",
			classlabels, nil,
		),
		requeues: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "requeues"),
			"Class requeues",
			classlabels, nil,
		),
		qlen: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "qlen"),
			"Class qlen",
			classlabels, nil,
		),
		bps: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "bps"),
			"Class bps",
			classlabels, nil,
		),
		pps: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "pps"),
			"Class pps",
			classlabels, nil,
		),
	}, nil
}

func (c *genericClassCollector) Update(ch chan<- prometheus.Metric) error {
	host, err := os.Hostname()
	if err != nil {
		logrus.Errorf("couldn't get host name: %v\n", err)
		return err
	}
	linkindex := strconv.Itoa(c.class.Attrs().LinkIndex)

	ch <- prometheus.MustNewConstMetric(
		c.bytes,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Basic.Bytes),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.packets,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Basic.Packets),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.backlog,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Queue.Backlog),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.drops,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Queue.Drops),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.overlimits,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Queue.Overlimits),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.requeues,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Queue.Requeues),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.qlen,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Queue.Qlen),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.bps,
		prometheus.GaugeValue,
		float64(c.class.Statistics.RateEst.Bps),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.pps,
		prometheus.GaugeValue,
		float64(c.class.Statistics.RateEst.Pps),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	return nil
}

// Collector for a hfsc class
type hfscClassCollector struct {
	link       string
	class      *netlink.HfscClass
	bytes      *prometheus.Desc
	packets    *prometheus.Desc
	backlog    *prometheus.Desc
	drops      *prometheus.Desc
	overlimits *prometheus.Desc
	requeues   *prometheus.Desc
	qlen       *prometheus.Desc
	bps        *prometheus.Desc
	pps        *prometheus.Desc
}

func NewHfscClassCollector(class netlink.Class, link string) (Collector, error) {
	return &hfscClassCollector{
		link:  link,
		class: class.(*netlink.HfscClass),
		bytes: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "bytes"),
			"Bytes passed though class",
			classlabels, nil,
		),
		packets: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "packets"),
			"Packets passed through class",
			classlabels, nil,
		),
		backlog: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "backlog"),
			"Class backlog",
			classlabels, nil,
		),
		drops: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "drops"),
			"Class drops",
			classlabels, nil,
		),
		overlimits: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "overlimits"),
			"Class overlimits",
			classlabels, nil,
		),
		requeues: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "requeues"),
			"Class requeues",
			classlabels, nil,
		),
		qlen: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "qlen"),
			"Class qlen",
			classlabels, nil,
		),
		bps: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "bps"),
			"Class bps",
			classlabels, nil,
		),
		pps: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "pps"),
			"Class pps",
			classlabels, nil,
		),
	}, nil
}

func (c *hfscClassCollector) Update(ch chan<- prometheus.Metric) error {
	host, err := os.Hostname()
	if err != nil {
		logrus.Errorf("couldn't get host name: %v\n", err)
		return err
	}
	linkindex := strconv.Itoa(c.class.Attrs().LinkIndex)

	ch <- prometheus.MustNewConstMetric(
		c.bytes,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Basic.Bytes),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.packets,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Basic.Packets),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.backlog,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Queue.Backlog),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.drops,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Queue.Drops),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.overlimits,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Queue.Overlimits),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.requeues,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Queue.Requeues),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.qlen,
		prometheus.GaugeValue,
		float64(c.class.Statistics.Queue.Qlen),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.bps,
		prometheus.GaugeValue,
		float64(c.class.Statistics.RateEst.Bps),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	ch <- prometheus.MustNewConstMetric(
		c.pps,
		prometheus.GaugeValue,
		float64(c.class.Statistics.RateEst.Pps),
		host,
		linkindex,
		c.link,
		c.class.Type(),
		netlink.HandleStr(c.class.Attrs().Handle),
		netlink.HandleStr(c.class.Attrs().Parent),
		netlink.HandleStr(c.class.Attrs().Leaf),
	)
	return nil
}
