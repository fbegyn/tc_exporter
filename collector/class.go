package tccollector

import (
	"fmt"
	"net"
	"os"

	"github.com/florianl/go-tc"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
)

var (
	classlabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
	curvelabels []string = []string{"host", "netns", "linkindex", "link", "type", "handle", "parent"}
)

type ClassCollector struct {
	logger     log.Logger
	netns      map[int][]*net.Interface
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

func NewClassCollector(netns map[int][]*net.Interface, clog log.Logger) (prometheus.Collector, error) {
	// Setup logger for qdisc collector
	clog = log.With(clog, "collector", "class")
	clog.Log("msg", "making class collector")

	return &ClassCollector{
		logger: clog,
		netns:  netns,
		bytes: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "bytes_total"),
			"Qdisc byte counter",
			classlabels, nil,
		),
		packets: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "packets_total"),
			"Qdisc packet counter",
			classlabels, nil,
		),
		bps: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "bps"),
			"Qdisc byte rate",
			classlabels, nil,
		),
		pps: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "pps"),
			"Qdisc packet rate",
			classlabels, nil,
		),
		backlog: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "backlog_total"),
			"Qdisc queue backlog",
			classlabels, nil,
		),
		drops: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "drops_total"),
			"Qdisc queue drops",
			classlabels, nil,
		),
		overlimits: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "overlimits_total"),
			"Qdisc queue overlimits",
			classlabels, nil,
		),
		qlen: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "qlen_total"),
			"Qdisc queue length",
			classlabels, nil,
		),
		requeues: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "requeque_total"),
			"Qdisc requeque counter",
			classlabels, nil,
		),
	}, nil
}

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
	}

	for _, d := range ds {
		ch <- d
	}
}

func (cc *ClassCollector) Collect(ch chan<- prometheus.Metric) {
	host, err := os.Hostname()
	if err != nil {
		cc.logger.Log("msg", "failed to get hostname", "err", err)
	}

	for ns, devices := range cc.netns {
		for _, interf := range devices {
			classes, err := getClasses(uint32(interf.Index), ns)
			if err != nil {
				cc.logger.Log("msg", "failed to get classes", "interface", interf.Name, "err", err)
			}

			for _, cl := range classes {
				handleMaj, handleMin := HandleStr(cl.Handle)
				parentMaj, parentMin := HandleStr(cl.Parent)

				ch <- prometheus.MustNewConstMetric(
					cc.bytes,
					prometheus.CounterValue,
					float64(cl.Stats.Bytes),
					host,
					fmt.Sprintf("%d", ns),
					fmt.Sprintf("%d", interf.Index),
					interf.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.packets,
					prometheus.CounterValue,
					float64(cl.Stats.Packets),
					host,
					fmt.Sprintf("%d", ns),
					fmt.Sprintf("%d", interf.Index),
					interf.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.bps,
					prometheus.GaugeValue,
					float64(cl.Stats.Bps),
					host,
					fmt.Sprintf("%d", ns),
					fmt.Sprintf("%d", interf.Index),
					interf.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.pps,
					prometheus.GaugeValue,
					float64(cl.Stats.Pps),
					host,
					fmt.Sprintf("%d", ns),
					fmt.Sprintf("%d", interf.Index),
					interf.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.backlog,
					prometheus.CounterValue,
					float64(cl.Stats.Backlog),
					host,
					fmt.Sprintf("%d", ns),
					fmt.Sprintf("%d", interf.Index),
					interf.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.drops,
					prometheus.CounterValue,
					float64(cl.Stats.Drops),
					host,
					fmt.Sprintf("%d", ns),
					fmt.Sprintf("%d", interf.Index),
					interf.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.overlimits,
					prometheus.CounterValue,
					float64(cl.Stats.Overlimits),
					host,
					fmt.Sprintf("%d", ns),
					fmt.Sprintf("%d", interf.Index),
					interf.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.qlen,
					prometheus.CounterValue,
					float64(cl.Stats.Qlen),
					host,
					fmt.Sprintf("%d", ns),
					fmt.Sprintf("%d", interf.Index),
					interf.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
				ch <- prometheus.MustNewConstMetric(
					cc.requeues,
					prometheus.CounterValue,
					float64(cl.Stats2.Requeues),
					host,
					fmt.Sprintf("%d", ns),
					fmt.Sprintf("%d", interf.Index),
					interf.Name,
					cl.Kind,
					fmt.Sprintf("%x:%x", handleMaj, handleMin),
					fmt.Sprintf("%x:%x", parentMaj, parentMin),
				)
			}

		}
	}

}

type ServiceCurveCollector struct {
	logger log.Logger
	netns  map[int][]*net.Interface
	curves map[string]*tc.ServiceCurve
	Burst  *prometheus.Desc
	Delay  *prometheus.Desc
	Rate   *prometheus.Desc
}

func NewServiceCurveCollector(netns map[int][]*net.Interface, sclog log.Logger) (prometheus.Collector, error) {

	sclog = log.With(sclog, "collector", "hfsc")
	sclog.Log("msg", "making SC collector")

	curves := make(map[string]*tc.ServiceCurve)

	return &ServiceCurveCollector{
		logger: sclog,
		curves: curves,
		netns:  netns,
		Burst: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "service_curve", "burst"),
			"Burst parameter of the service curve",
			curvelabels, nil,
		),
		Delay: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "service_curve", "delay"),
			"Delay parameter of the service curve",
			curvelabels, nil,
		),
		Rate: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "service_curve", "rate"),
			"Rate parameter of the service curve",
			curvelabels, nil,
		),
	}, nil
}

func (c *ServiceCurveCollector) Describe(ch chan<- *prometheus.Desc) {
	ds := []*prometheus.Desc{
		c.Burst,
		c.Delay,
		c.Rate,
	}

	for _, d := range ds {
		ch <- d
	}
}

//curvelabels []string = []string{"host", "linkindex", "link", "type", "handle", "parent", "leaf"}
func (c *ServiceCurveCollector) Collect(ch chan<- prometheus.Metric) {
	host, err := os.Hostname()
	if err != nil {
		c.logger.Log("msg", "failed to get hostname", "err", err)
	}

	for ns, devices := range c.netns {
		for _, interf := range devices {
			classes, err := getClasses(uint32(interf.Index), ns)
			if err != nil {
				c.logger.Log("msg", "failed to get classes", "interface", interf.Name, "err", err)
			}

			for _, cl := range classes {
				handleMaj, handleMin := HandleStr(cl.Handle)
				parentMaj, parentMin := HandleStr(cl.Parent)

				if cl.Hfsc != nil {
					c.curves["fsc"] = cl.Hfsc.Fsc
					c.curves["rsc"] = cl.Hfsc.Rsc
					c.curves["usc"] = cl.Hfsc.Usc
				}

				for typ, sc := range c.curves {
					if sc == nil {
						continue
					}
					ch <- prometheus.MustNewConstMetric(
						c.Burst,
						prometheus.GaugeValue,
						float64(sc.M1),
						host,
						fmt.Sprintf("%d", ns),
						fmt.Sprintf("%d", interf.Index),
						interf.Name,
						typ,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
					ch <- prometheus.MustNewConstMetric(
						c.Delay,
						prometheus.GaugeValue,
						float64(sc.D),
						host,
						fmt.Sprintf("%d", ns),
						fmt.Sprintf("%d", interf.Index),
						interf.Name,
						typ,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
					ch <- prometheus.MustNewConstMetric(
						c.Rate,
						prometheus.GaugeValue,
						float64(sc.M2),
						host,
						fmt.Sprintf("%d", ns),
						fmt.Sprintf("%d", interf.Index),
						interf.Name,
						typ,
						fmt.Sprintf("%x:%x", handleMaj, handleMin),
						fmt.Sprintf("%x:%x", parentMaj, parentMin),
					)
				}
			}
		}
	}

}

func getClasses(devid uint32, ns int) ([]tc.Object, error) {
	// Create socket for interface to get classes from
	sock, err := tc.Open(&tc.Config{
		NetNS: ns,
	})
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := sock.Close(); err != nil {
		}
	}()
	classes, err := sock.Class().Get(&tc.Msg{
		Family:  unix.AF_UNSPEC,
		Ifindex: devid,
	})
	if err != nil {
		return nil, err
	}
	var cl []tc.Object
	for _, class := range classes {
		if class.Ifindex == devid && class.Kind != "fq_codel" {
			cl = append(cl, class)
		}
	}
	return cl, nil
}
