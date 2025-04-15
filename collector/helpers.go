package tccollector

import (
	"os"
	"fmt"

	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/mdlayher/netlink"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sys/unix"
)

// HandleStr is a simple helper function that construct human readable handles
func HandleStr(handle uint32) (uint32, uint32) {
	return ((handle & 0xffff0000) >> 16), (handle & 0x0000ffff)
}

// HandleStr is a simple helper function that construct human readable handles
func FmtHandleStr(handle uint32) (string) {
	return fmt.Sprintf("%d:%d", ((handle & 0xffff0000) >> 16), (handle & 0x0000ffff))
}

// GetNetlinkConn gets a rtnetlink connection for the specified network namespace
func GetNetlinkConn(ns string) (con *rtnetlink.Conn, err error) {
	if ns == "default" {
		con, err = rtnetlink.Dial(nil)
		if err != nil {
			return nil, err
		}
	} else {
		f, err := os.Open("/var/run/netns/" + ns)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		con, err = rtnetlink.Dial(&netlink.Config{
			NetNS: int(f.Fd()),
		})
		if err != nil {
			return nil, err
		}
	}
	return
}

// GetTcConn gets a TC connection for the specifed network namespace
func GetTcConn(ns string) (sock *tc.Tc, err error) {
	if ns == "default" {
		sock, err = tc.Open(&tc.Config{})
		if err != nil {
			return nil, err
		}
	} else {
		f, err := os.Open("/var/run/netns/" + ns)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		sock, err = tc.Open(&tc.Config{
			NetNS: int(f.Fd()),
		})
		if err != nil {
			return nil, err
		}
	}
	return
}

// getQdiscs fetches all qdiscs for a pecified interface in the netns
func getQdiscs(devid uint32, ns string) ([]tc.Object, error) {
	sock, err := GetTcConn(ns)
	if err != nil {
		return nil, err
	}
	defer sock.Close()
	qdiscs, err := sock.Qdisc().Get()
	if err != nil {
		return nil, err
	}
	var qd []tc.Object
	for _, qdisc := range qdiscs {
		if qdisc.Ifindex == devid {
			qd = append(qd, qdisc)
		}
	}
	return qd, nil
}

// getQdiscs fetches all qdiscs for a pecified interface in the netns
func getClasses(devid uint32, ns string) ([]tc.Object, error) {
	sock, err := GetTcConn(ns)
	if err != nil {
		return nil, err
	}
	defer sock.Close()

	classes, err := sock.Class().Get(&tc.Msg{
		Family:  unix.AF_UNSPEC,
		Info:    0,
		Handle:  tc.HandleRoot,
		Ifindex: devid,
	})
	if err != nil {
		return nil, err
	}
	var cl []tc.Object
	for _, class := range classes {
		if class.Ifindex == devid {
			cl = append(cl, class)
		}
	}
	return cl, nil
}

// getQdiscs fetches all qdiscs for a pecified interface in the netns
func getFilters(devid uint32, ns string) ([]tc.Object, error) {
	sock, err := GetTcConn(ns)
	if err != nil {
		return nil, err
	}
	defer sock.Close()

	filters, err := sock.Filter().Get(&tc.Msg{
		Family:  unix.AF_UNSPEC,
		Info:    0,
		Handle:  tc.HandleRoot,
		Ifindex: devid,
	})
	if err != nil {
		return nil, err
	}
	var fl []tc.Object
	for _, filter := range filters {
		if filter.Ifindex == devid {
			fl = append(fl, filter)
		}
	}

	return fl, nil
}

type stats struct {
	bytes      *prometheus.Desc
	packets    *prometheus.Desc
	drops      *prometheus.Desc
	overlimits *prometheus.Desc
	bps        *prometheus.Desc
	pps        *prometheus.Desc
	qlen       *prometheus.Desc
	backlog    *prometheus.Desc
	requeues   *prometheus.Desc
}
