package main

import (
	"os"
	"time"
  "strconv"

  "github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
	"github.com/influxdata/influxdb/client/v2"
)

// WriteLink writes te data from a link to influx
func WriteLink(dbclient client.Client, db string, l netlink.Link) {
	host, _ := os.Hostname()

	batch, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: db,
	})
	if err != nil {
		logrus.Errorf("Failed to create netlink link batch points: %v\n.", err)
	}

  tags := map[string]string{
    "host": host,
    "name": l.Attrs().Name,
    "type": l.Type(),
    "hwaddr": l.Attrs().HardwareAddr.String(),
  }

  state := 0
  if l.Attrs().OperState.String() == "up" {
    state = 1
  }

  fields := map[string]interface{}{
    "operstate": state,
  }

  influxPoint, err := client.NewPoint("link_stats", tags, fields, time.Now())
  if err != nil {
    logrus.Errorf("Failed to make a new link_stats point: %v.\n", err)
  }
  batch.AddPoint(influxPoint)

	err = dbclient.Write(batch)
	if err != nil {
		logrus.Errorf("Failed to write link_stats point to influx: %v.\n", err)
	}
}

// WriteQdisc writes the data from a qdisc to influx
func WriteQdisc(dbclient client.Client, db string, qdiscs *[]netlink.Qdisc) {
	host, _ := os.Hostname()

	batch, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: db,
	})
	if err != nil {
		logrus.Errorf("Failed to create netlink qdisc batch points: %v\n.", err)
	}

  for _, q := range *qdiscs {
    tags := map[string]string{
      "host": host,
      "linkindex": strconv.Itoa(q.Attrs().LinkIndex),
      "type": q.Type(),
      "handle": netlink.HandleStr(q.Attrs().Handle),
      "parent": netlink.HandleStr(q.Attrs().Parent),
    }

    fields := map[string]interface{}{
      "refcnt": q.Attrs().Refcnt,
    }

    switch q.Type() {
      case "hfsc":
        qd := q.(*netlink.Hfsc)
        fields["default"] = qd.Defcls
      case "fq_codel":
        qd := q.(*netlink.FqCodel)
        fields["target"] = qd.Target
        fields["limit"] = qd.Limit
        fields["interval"] = qd.Interval
        fields["ecn"] = qd.ECN
        fields["flows"] = qd.Flows
        fields["quantum"] = qd.Quantum
    }
    influxPoint, err := client.NewPoint("qdisc_stats", tags, fields, time.Now())
    if err != nil {
      logrus.Errorf("Failed to make a new qdisc point: %v.\n", err)
    }
    batch.AddPoint(influxPoint)
  }

  err = dbclient.Write(batch)
	if err != nil {
		logrus.Errorf("Failed to write qdisc point to influx: %v.\n", err)
	}
}

// WriteClass writes the data to an influx db
func WriteClass(dbclient client.Client, db string, classes *[]netlink.Class) {
	host, _ := os.Hostname()

	batch, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: db,
	})
	if err != nil {
		logrus.Errorf("Failed to create netlink class batch points: %v\n.", err)
	}

  for _, c := range *classes{
    tags := map[string]string{
      "host": host,
      "linkindex": strconv.Itoa(c.Attrs().LinkIndex),
      "type": c.Type(),
      "handle": netlink.HandleStr(c.Attrs().Handle),
      "parent": netlink.HandleStr(c.Attrs().Parent),
      "leaf": strconv.FormatUint(uint64(c.Attrs().Leaf), 10),
    }

    fields := map[string]interface{}{
      "bytes": uint32(c.Attrs().Statistics.Basic.Bytes),
      "packets": c.Attrs().Statistics.Basic.Packets,
      "backlog": c.Attrs().Statistics.Queue.Backlog,
      "drops": c.Attrs().Statistics.Queue.Drops,
      "overlimits": c.Attrs().Statistics.Queue.Overlimits,
      "requeues": c.Attrs().Statistics.Queue.Requeues,
      "qlen": c.Attrs().Statistics.Queue.Qlen,
      "bps": c.Attrs().Statistics.RateEst.Bps,
      "pps": c.Attrs().Statistics.RateEst.Pps,
    }

    switch c.Type() {
      case "hfsc":
        cd := c.(*netlink.HfscClass)
        Fburst, Fdelay, Frate := cd.Fsc.Attrs()
        Uburst, Udelay, Urate := cd.Usc.Attrs()
        Rburst, Rdelay, Rrate := cd.Rsc.Attrs()
        fields["fsc_burst"] = Fburst
        fields["fsc_delay"] = Fdelay
        fields["fsc_rate"] = Frate
        fields["rsc_burst"] = Rburst
        fields["rsc_delay"] = Rdelay
        fields["rsc_rate"] = Rrate
        fields["usc_burst"] = Uburst
        fields["usc_delay"] = Udelay
        fields["usc_rate"] = Urate
    }
    influxPoint, err := client.NewPoint("class_stats", tags, fields, time.Now())
    if err != nil {
      logrus.Errorf("Failed to make a new class point: %v.\n", err)
    }
    batch.AddPoint(influxPoint)
  }

  err = dbclient.Write(batch)
	if err != nil {
		logrus.Errorf("Failed to write classes point to influx: %v.\n", err)
	}
}
