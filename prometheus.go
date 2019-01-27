package main

import (
        "github.com/prometheus/client_golang/prometheus"
        "github.com/prometheus/client_golang/prometheus/promhttp"
        "github.com/sirupsen/logrus"
        "github.com/vishvananda/netlink"
        "net/http"
        "os"
        "strconv"
)

var (
        operstatus = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_link_operstatus",
                        Help: "Operational status of the link",
                },
                []string{"host", "name", "type", "hwaddr"},
        )
        qdiscRefcnt = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_qdisc_refcnt",
                        Help: "Qdisc refcount",
                },
                []string{"host", "linkindex", "type", "handle", "parent"},
        )
        hfscDefault = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_qdisc_hfsc_default",
                        Help: "Default class id for the HFSC qdisc",
                },
                []string{"host", "linkindex", "type", "handle", "parent"},
        )
        fqcodelTarget = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_qdisc_fqcodel_target",
                        Help: "The acceptable minimum standing/persistent queue delay",
                },
                []string{"host", "linkindex", "type", "handle", "parent"},
        )
        fqcodelLimit = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_qdisc_fqcodel_limit",
                        Help: "The hard limit on the real queue size",
                },
                []string{"host", "linkindex", "type", "handle", "parent"},
        )
        fqcodelInterval = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_qdisc_fqcodel_interval",
                        Help: "Used to ensure that the measured minimum delay does not become too stale",
                },
                []string{"host", "linkindex", "type", "handle", "parent"},
        )
        fqcodelECN = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_qdisc_fqcodel_ecn",
                        Help: "Can be used to mark packets instead of dropping them<Paste>",
                },
                []string{"host", "linkindex", "type", "handle", "parent"},
        )
        fqcodelFlows = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_qdisc_fqcodel_flows",
                        Help: "The number of flows into which the incoming packets are classified",
                },
                []string{"host", "linkindex", "type", "handle", "parent"},
        )
        fqcodelQuantum = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_qdisc_fqcodel_quantum",
                        Help: "The number of bytes used as 'deficit' in the fair queuing algorithm",
                },
                []string{"host", "linkindex", "type", "handle", "parent"},
        )
        classBytes = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_class_bytes",
                        Help: "Sent bytes",
                },
                []string{"host", "linkindex", "type", "handle", "parent", "leaf"},
        )
        classPackets = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_class_packets",
                        Help: "Sent packets",
                },
                []string{"host", "linkindex", "type", "handle", "parent", "leaf"},
        )
        classBacklog = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_class_backlog",
                        Help: "Packets in backlog",
                },
                []string{"host", "linkindex", "type", "handle", "parent", "leaf"},
        )
        classDrops = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_class_drops",
                        Help: "Dropped packets",
                },
                []string{"host", "linkindex", "type", "handle", "parent", "leaf"},
        )
        classOverlimits = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_class_overlimits",
                        Help: "Overlimit packets",
                },
                []string{"host", "linkindex", "type", "handle", "parent", "leaf"},
        )
        classRequeues = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_class_requeues",
                        Help: "Requeue packets",
                },
                []string{"host", "linkindex", "type", "handle", "parent", "leaf"},
        )
        classQlen = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_class_qlen",
                        Help: "Packets in qlen",
                },
                []string{"host", "linkindex", "type", "handle", "parent", "leaf"},
        )
        classBps = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_class_bps",
                        Help: "Bytes per second",
                },
                []string{"host", "linkindex", "type", "handle", "parent", "leaf"},
        )
        classPps = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_class_pps",
                        Help: "Packets per second",
                },
                []string{"host", "linkindex", "type", "handle", "parent", "leaf"},
        )
        hfscSC = prometheus.NewGaugeVec(
                prometheus.GaugeOpts{
                        Name: "tc_class_hfsc_sc",
                        Help: "Service curve for the fhsc class,",
                },
                []string{"host", "linkindex", "type", "handle", "parent", "leaf", "sc", "param"},
        )
)

func init() {
        prometheus.MustRegister(operstatus)
        prometheus.MustRegister(qdiscRefcnt)
        prometheus.MustRegister(hfscDefault)
        prometheus.MustRegister(fqcodelTarget)
        prometheus.MustRegister(fqcodelLimit)
        prometheus.MustRegister(fqcodelQuantum)
        prometheus.MustRegister(fqcodelFlows)
        prometheus.MustRegister(fqcodelECN)
        prometheus.MustRegister(fqcodelInterval)
        prometheus.MustRegister(classBytes)
        prometheus.MustRegister(classPackets)
        prometheus.MustRegister(classBacklog)
        prometheus.MustRegister(classDrops)
        prometheus.MustRegister(classOverlimits)
        prometheus.MustRegister(classRequeues)
        prometheus.MustRegister(classQlen)
        prometheus.MustRegister(classBps)
        prometheus.MustRegister(classPps)
        prometheus.MustRegister(hfscSC)
}

func HandleProm(link *netlink.Link, qdiscs *[]netlink.Qdisc, classes *[]netlink.Class) {
  go registerLink(*link)
  go registerQdiscs(qdiscs)
  go registerClasses(classes)
}

func PromExporter(port string) {
  logrus.Infoln("Starting prometheus exporter on http://localhost:9601/metrics")
  http.Handle("/metrics", promhttp.Handler())
  logrus.Fatal(http.ListenAndServe(port, nil))
}

func registerLink(l netlink.Link){
  host, err := os.Hostname()
  if err != nil {
    logrus.Errorf("couldn't get host name: %v\n", err)
  }
  if l.Attrs().OperState.String() == "up" {
    operstatus.WithLabelValues(host, l.Attrs().Name, l.Type(), l.Attrs().HardwareAddr.String()).Set(1)
  } else {
    operstatus.WithLabelValues(host, l.Attrs().Name, l.Type(), l.Attrs().HardwareAddr.String()).Set(0)
  }
}

func registerQdiscs(qdiscs *[]netlink.Qdisc) {
  for _, q := range *qdiscs {
    registerQdisc(q)
  }
}

func registerQdisc(q netlink.Qdisc) {
  host, err := os.Hostname()
  if err != nil {
    logrus.Errorf("couldn't get host name: %v\n", err)
  }
  linkindex := strconv.Itoa(q.Attrs().LinkIndex)
  typ := q.Type()
  handle := netlink.HandleStr(q.Attrs().Handle)
  parent := netlink.HandleStr(q.Attrs().Parent)
  qdiscRefcnt.WithLabelValues(host, linkindex, typ, handle, parent).Set(float64(q.Attrs().Refcnt))
  switch typ {
    case "hfsc":
      qd := q.(*netlink.Hfsc)
      hfscDefault.WithLabelValues(host, linkindex, typ, handle, parent).Set(float64(qd.Defcls))
    case "fq_codel":
      qd := q.(*netlink.FqCodel)
      fqcodelTarget.WithLabelValues(host, linkindex, typ, handle, parent).Set(float64(qd.Target+1))
      fqcodelLimit.WithLabelValues(host, linkindex, typ, handle, parent).Set(float64(qd.Limit))
      fqcodelInterval.WithLabelValues(host, linkindex, typ, handle, parent).Set(float64(qd.Interval+1))
      fqcodelECN.WithLabelValues(host, linkindex, typ, handle, parent).Set(float64(qd.ECN))
      fqcodelFlows.WithLabelValues(host, linkindex, typ, handle, parent).Set(float64(qd.Flows))
      fqcodelQuantum.WithLabelValues(host, linkindex, typ, handle, parent).Set(float64(qd.Quantum))
  }
}

func registerClasses(classes *[]netlink.Class) {
  for _, c := range *classes {
    registerClass(c)
  }
}

func registerClass(c netlink.Class) {
  host, err := os.Hostname()
  if err != nil {
    logrus.Errorf("couldn't get host name: %v\n", err)
  }
  linkindex := strconv.Itoa(c.Attrs().LinkIndex)
  typ := c.Type()
  handle := netlink.HandleStr(c.Attrs().Handle)
  parent := netlink.HandleStr(c.Attrs().Parent)
  leaf := netlink.HandleStr(c.Attrs().Leaf)
  classBytes.WithLabelValues(host, linkindex, typ, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Basic.Bytes))
  classPackets.WithLabelValues(host, linkindex, typ, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Basic.Packets))
  classBacklog.WithLabelValues(host, linkindex, typ, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Queue.Backlog))
  classDrops.WithLabelValues(host, linkindex, typ, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Queue.Drops))
  classOverlimits.WithLabelValues(host, linkindex, typ, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Queue.Overlimits))
  classRequeues.WithLabelValues(host, linkindex, typ, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Queue.Requeues))
  classQlen.WithLabelValues(host, linkindex, typ, handle, parent, leaf).Set(float64(c.Attrs().Statistics.Queue.Qlen))
  classBps.WithLabelValues(host, linkindex, typ, handle, parent, leaf).Set(float64(c.Attrs().Statistics.RateEst.Bps))
  classPps.WithLabelValues(host, linkindex, typ, handle, parent, leaf).Set(float64(c.Attrs().Statistics.RateEst.Pps))
  switch typ {
    case "hfsc":
      cd := c.(*netlink.HfscClass)
      Fburst, Fdelay, Frate := cd.Fsc.Attrs()
      Uburst, Udelay, Urate := cd.Usc.Attrs()
      Rburst, Rdelay, Rrate := cd.Rsc.Attrs()
      hfscSC.WithLabelValues(host, linkindex, typ, handle, parent, leaf, "fsc", "burst").Set(float64(Fburst))
      hfscSC.WithLabelValues(host, linkindex, typ, handle, parent, leaf, "fsc", "delay").Set(float64(Fdelay))
      hfscSC.WithLabelValues(host, linkindex, typ, handle, parent, leaf, "fsc", "rate").Set(float64(Frate))
      hfscSC.WithLabelValues(host, linkindex, typ, handle, parent, leaf, "rsc", "burst").Set(float64(Rburst))
      hfscSC.WithLabelValues(host, linkindex, typ, handle, parent, leaf, "rsc", "delay").Set(float64(Rdelay))
      hfscSC.WithLabelValues(host, linkindex, typ, handle, parent, leaf, "rsc", "rate").Set(float64(Rrate))
      hfscSC.WithLabelValues(host, linkindex, typ, handle, parent, leaf, "usc", "burst").Set(float64(Uburst))
      hfscSC.WithLabelValues(host, linkindex, typ, handle, parent, leaf, "usc", "delay").Set(float64(Udelay))
      hfscSC.WithLabelValues(host, linkindex, typ, handle, parent, leaf, "usc", "rate").Set(float64(Urate))
  }
}
