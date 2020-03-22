package collector

import (
	"github.com/florianl/go-tc"
	"github.com/go-kit/kit/log"
	"golang.org/x/sys/unix"
)

// HandleStr is a simple helper function that cinstruct human readable handles
func HandleStr(handle uint32) (uint32, uint32) {
	return ((handle & 0xffff0000) >> 16), (handle & 0x0000ffff)
}

func GetNetlinkData(sock *tc.Tc, devid uint32, logger log.Logger) ([]tc.Object, []tc.Object) {
	return GetQdiscs(sock, devid, logger), GetClasses(sock, devid, logger)
}

func GetQdiscs(sock *tc.Tc, devid uint32, logger log.Logger) []tc.Object {
	qdiscs, err := sock.Qdisc().Get()
	if err != nil {
		logger.Log("msg", "failed to get qdiscs", "err", err)
	}
	var qd []tc.Object
	for _, qdisc := range qdiscs {
		if qdisc.Ifindex == devid {
			qd = append(qd, qdisc)
		}
	}
	return qd
}

func GetClasses(sock *tc.Tc, devid uint32, logger log.Logger) []tc.Object {
	classes, err := sock.Class().Get(&tc.Msg{
		Family:  unix.AF_UNSPEC,
		Ifindex: devid,
	})
	if err != nil {
		logger.Log("msg", "failed to get classes", "err", err)
	}
	var cl []tc.Object
	for _, class := range classes {
		if class.Ifindex == devid && class.Kind != "fq_codel" {
			cl = append(cl, class)
		}
	}
	return cl
}
