package collector

import (
	"fmt"
	"net"
	"os"

	"github.com/florianl/go-tc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	classlabels []string = []string{"host", "linkindex", "link", "type", "handle", "parent"}
	curvelabels []string = []string{"host", "linkindex", "link", "type", "handle", "parent"}
)

type ClassCollector struct {
	interf     string
	devID      uint32
	class      tc.Object
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

func NewClassCollector(interf *net.Interface, class tc.Object) (Collector, error) {
	return &ClassCollector{
		interf: interf.Name,
		devID:  uint32(interf.Index),
		class:  class,
		bytes: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "bytes"),
			"Qdisc byte counter",
			classlabels, nil,
		),
		packets: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "packets"),
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
			prometheus.BuildFQName(namespace, "class", "backlog"),
			"Qdisc queue backlog",
			classlabels, nil,
		),
		drops: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "drops"),
			"Qdisc queue drops",
			classlabels, nil,
		),
		overlimits: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "overlimits"),
			"Qdisc queue overlimits",
			classlabels, nil,
		),
		qlen: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "qlen"),
			"Qdisc queue length",
			classlabels, nil,
		),
		requeues: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "class", "requeque"),
			"Qdisc requeque counter",
			classlabels, nil,
		),
	}, nil
}

func (cc *ClassCollector) Update(ch chan<- prometheus.Metric) error {
	host, err := os.Hostname()
	if err != nil {
		logrus.Errorf("couldn't get host name: %v\n", err)
		return err
	}

	handleMaj, handleMin := HandleStr(cc.class.Handle)
	parentMaj, parentMin := HandleStr(cc.class.Parent)

	ch <- prometheus.MustNewConstMetric(
		cc.bytes,
		prometheus.CounterValue,
		float64(cc.class.Stats.Bytes),
		host,
		fmt.Sprintf("%d", cc.devID),
		cc.interf,
		cc.class.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		cc.packets,
		prometheus.CounterValue,
		float64(cc.class.Stats.Packets),
		host,
		fmt.Sprintf("%d", cc.devID),
		cc.interf,
		cc.class.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		cc.bps,
		prometheus.CounterValue,
		float64(cc.class.Stats.Bps),
		host,
		fmt.Sprintf("%d", cc.devID),
		cc.interf,
		cc.class.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		cc.pps,
		prometheus.CounterValue,
		float64(cc.class.Stats.Pps),
		host,
		fmt.Sprintf("%d", cc.devID),
		cc.interf,
		cc.class.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		cc.backlog,
		prometheus.CounterValue,
		float64(cc.class.Stats.Backlog),
		host,
		fmt.Sprintf("%d", cc.devID),
		cc.interf,
		cc.class.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		cc.drops,
		prometheus.CounterValue,
		float64(cc.class.Stats.Drops),
		host,
		fmt.Sprintf("%d", cc.devID),
		cc.interf,
		cc.class.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		cc.overlimits,
		prometheus.CounterValue,
		float64(cc.class.Stats.Overlimits),
		host,
		fmt.Sprintf("%d", cc.devID),
		cc.interf,
		cc.class.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		cc.qlen,
		prometheus.CounterValue,
		float64(cc.class.Stats.Qlen),
		host,
		fmt.Sprintf("%d", cc.devID),
		cc.interf,
		cc.class.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		cc.requeues,
		prometheus.CounterValue,
		float64(cc.class.Stats2.Requeues),
		host,
		fmt.Sprintf("%d", cc.devID),
		cc.interf,
		cc.class.Kind,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	return nil
}

type HfscCollector struct {
	interf string
	devID  uint32
	class  tc.Object
	usc    Collector
	rsc    Collector
	fsc    Collector
}

func NewHfscCollector(interf *net.Interface, class tc.Object) (Collector, error) {

	FSC, _ := newServiceCurveCollector(class, class.Hfsc.Fsc, "fsc")
	RSC, _ := newServiceCurveCollector(class, class.Hfsc.Fsc, "usc")
	USC, _ := newServiceCurveCollector(class, class.Hfsc.Fsc, "rsc")

	return &HfscCollector{
		interf: interf.Name,
		devID:  uint32(interf.Index),
		class:  class,
		usc:    USC,
		rsc:    RSC,
		fsc:    FSC,
	}, nil
}

func (c *HfscCollector) Update(ch chan<- prometheus.Metric) error {
	c.usc.Update(ch)
	c.rsc.Update(ch)
	c.fsc.Update(ch)
	return nil
}

type serviceCurveCollector struct {
	interf string
	sc     string
	devID  uint32
	class  tc.Object
	curve  *tc.ServiceCurve
	Burst  *prometheus.Desc
	Delay  *prometheus.Desc
	Rate   *prometheus.Desc
}

func newServiceCurveCollector(class tc.Object, curve *tc.ServiceCurve, sc string) (Collector, error) {

	interf, err := net.InterfaceByIndex(int(class.Ifindex))
	if err != nil {
		return nil, err
	}

	return &serviceCurveCollector{
		interf: interf.Name,
		sc:     sc,
		devID:  class.Ifindex,
		class:  class,
		curve:  curve,
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

//curvelabels []string = []string{"host", "linkindex", "link", "type", "handle", "parent", "leaf"}
func (c *serviceCurveCollector) Update(ch chan<- prometheus.Metric) error {
	host, err := os.Hostname()
	if err != nil {
		logrus.Errorf("couldn't get host name: %v\n", err)
		return err
	}

	handleMaj, handleMin := HandleStr(c.class.Handle)
	parentMaj, parentMin := HandleStr(c.class.Parent)

	ch <- prometheus.MustNewConstMetric(
		c.Burst,
		prometheus.GaugeValue,
		float64(c.curve.M1),
		host,
		fmt.Sprintf("%d", c.devID),
		c.interf,
		c.sc,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		c.Delay,
		prometheus.GaugeValue,
		float64(c.curve.D),
		host,
		fmt.Sprintf("%d", c.devID),
		c.interf,
		c.sc,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	ch <- prometheus.MustNewConstMetric(
		c.Rate,
		prometheus.GaugeValue,
		float64(c.curve.M2),
		host,
		fmt.Sprintf("%d", c.devID),
		c.interf,
		c.sc,
		fmt.Sprintf("%x:%x", handleMaj, handleMin),
		fmt.Sprintf("%x:%x", parentMaj, parentMin),
	)
	return nil
}
