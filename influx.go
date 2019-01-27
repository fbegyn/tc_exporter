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
func WriteQdisc(dbclient client.Client, db string, q netlink.Qdisc) {
	host, _ := os.Hostname()

	batch, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: db,
	})
	if err != nil {
		logrus.Errorf("Failed to create netlink qdisc batch points: %v\n.", err)
	}

	tags := map[string]string{
		"host": host,
    "linkindex": strconv.Itoa(q.Attrs().LinkIndex),
    "handle": netlink.HandleStr(q.Attrs().Handle),
    "parent": netlink.HandleStr(q.Attrs().Parent),
	}

  switch q.Type() {
    case "hfsc":
      qd := q.(*netlink.Hfsc)
      fields := map[string]interface{}{
        "default": qd.Defcls,
      }
      influxPoint, err := client.NewPoint("hfsc_qdisc", tags, fields, time.Now())
      if err != nil {
        logrus.Errorf("Failed to make a new hfsc qdisc point: %v.\n", err)
      }
      batch.AddPoint(influxPoint)
    case "fq_codel":
      qd := q.(*netlink.FqCodel)
      fields := map[string]interface{}{
        "target": qd.Target,
        "limit": qd.Limit,
        "interval": qd.Interval,
        "ecn": qd.ECN,
        "flows": qd.Flows,
        "quantum": qd.Quantum,
      }
      influxPoint, err := client.NewPoint("fq_codel_qdisc", tags, fields, time.Now())
      if err != nil {
        logrus.Errorf("Failed to make a new fq_codel qdisc point: %v.\n", err)
      }
      batch.AddPoint(influxPoint)
    default:
      fields := map[string]interface{}{
        "temp": 1,
      }
      influxPoint, err := client.NewPoint("qdisc", tags, fields, time.Now())
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
func WriteClass(dbclient client.Client, db string, c netlink.Class) {
	host, _ := os.Hostname()

	batch, err := client.NewBatchPoints(client.BatchPointsConfig{
		Database: db,
	})
	if err != nil {
		logrus.Errorf("Failed to create netlink class batch points: %v\n.", err)
	}

	tags := map[string]string{
		"host": host,
    "linkindex": strconv.Itoa(c.Attrs().LinkIndex),
    "handle": netlink.HandleStr(c.Attrs().Handle),
    "parent": netlink.HandleStr(c.Attrs().Parent),
    "leaf": strconv.FormatUint(uint64(c.Attrs().Leaf), 10),
	}

  switch c.Type() {
    case "hfsc":
      cd := c.(*netlink.HfscClass)
      Fburst, Fdelay, Frate := cd.Fsc.Attrs()
      Uburst, Udelay, Urate := cd.Usc.Attrs()
      Rburst, Rdelay, Rrate := cd.Rsc.Attrs()
      Ffields := map[string]interface{}{
        "burst": Fburst,
        "delay": Fdelay,
        "rate":  Frate,
      }
      influxPoint, err := client.NewPoint("hfsc_class_fsc", tags, Ffields, time.Now())
      if err != nil {
        logrus.Errorf("Failed to make a new hfsc class fsc point: %v.\n", err)
      }
      batch.AddPoint(influxPoint)
      Rfields := map[string]interface{}{
        "burst": Rburst,
        "delay": Rdelay,
        "rate":  Rrate,
      }
      influxPoint, err = client.NewPoint("hfsc_class_rsc", tags, Rfields, time.Now())
      if err != nil {
        logrus.Errorf("Failed to make a new hfsc class rsc point: %v.\n", err)
      }
      batch.AddPoint(influxPoint)
      Ufields := map[string]interface{}{
        "burst": Uburst,
        "delay": Udelay,
        "rate":  Urate,
      }
      influxPoint, err = client.NewPoint("hfsc_class_usc", tags, Ufields, time.Now())
      if err != nil {
        logrus.Errorf("Failed to make a new hfscl class usc point: %v.\n", err)
      }
      batch.AddPoint(influxPoint)
    default:
      fields := map[string]interface{}{
        "bytes": c.Attrs().Statistics.Basic.Bytes,
        "packets": c.Attrs().Statistics.Basic.Packets,
        "backlog": c.Attrs().Statistics.Queue.Backlog,
        "drops": c.Attrs().Statistics.Queue.Drops,
        "overlimits": c.Attrs().Statistics.Queue.Overlimits,
        "requeues": c.Attrs().Statistics.Queue.Requeues,
        "qlen": c.Attrs().Statistics.Queue.Qlen,
        "bps": c.Attrs().Statistics.RateEst.Bps,
        "pps": c.Attrs().Statistics.RateEst.Pps,
      }
      influxPoint, err := client.NewPoint("class", tags, fields, time.Now())
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
