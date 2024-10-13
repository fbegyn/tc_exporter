package tccollector

import (
	"net"
	"os"

	"github.com/florianl/go-tc"
	"github.com/florianl/go-tc/internal/unix"
	"github.com/jsimonetti/rtnetlink"
	"github.com/mdlayher/netlink"
)

// HandleStr is a simple helper function that cinstruct human readable handles
func HandleStr(handle uint32) (uint32, uint32) {
	return ((handle & 0xffff0000) >> 16), (handle & 0x0000ffff)
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

	interf, err := net.InterfaceByName(ns)

	qdiscs, err := sock.Class().Get(&tc.Msg{
		Family: unix.AF_UNSPEC,
		Info: 0,
		Handle: tc.HandleRoot,
		Ifindex: net.,
	})
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
func getFilters(devid uint32, ns string) ([]tc.Object, error) {
	sock, err := GetTcConn(ns)
	if err != nil {
		return nil, err
	}
	defer sock.Close()
	qdiscs, err := sock.Filter().Get()
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
