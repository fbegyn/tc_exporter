package main

import (
	"fmt"
	"math"

	"github.com/sirupsen/logrus"
	"github.com/vishvananda/netlink"
)

// rates stores the calculated rates for ease of access
type rates struct {
	MaxRate uint32 `json:"maxrate"`
	RateC1  uint32 `json:"ratec1"`
	RateC2  uint32 `json:"ratec2"`
	RateC11 uint32 `json:"ratec11"`
	RateC12 uint32 `json:"ratec12"`
	RateC13 uint32 `json:"ratec13"`
	RateC21 uint32 `json:"ratec21"`
	RateC31 uint32 `json:"ratec31"`
	RateC32 uint32 `json:"ratec32"`
	RateC22 uint32 `json:"ratec22"`
	RateC23 uint32 `json:"ratec23"`
	RateC3  uint32 `json:"ratec3"`
}

// MapQdisc maps qdics to handles
func MapQdisc(qds []netlink.Qdisc, m map[string]*netlink.Qdisc) {
	for _, qd := range qds {
		m[netlink.HandleStr(qd.Attrs().Handle)] = &qd
	}
}

// MapClass maps classes to handles
func MapClass(cls []netlink.Class, m map[string]*netlink.HfscClass) {
	for _, cl := range cls {
		if cl.Type() == "hfsc" {
			m[netlink.HandleStr(cl.Attrs().Handle)] = cl.(*netlink.HfscClass)
		}
	}
}

// BitsToSI converts uint to SI units
func BitsToSI(b uint32) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int32(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cbps", float64(b)/float64(div), "kMGTPE"[exp])
}

// GetData Return the data for the selected link
func GetData(index string) (link netlink.Link, qdiscs []netlink.Qdisc, classes []netlink.Class) {
	links, err := netlink.LinkList()
	if err != nil {
		logrus.Fatalf("Failed to fetch link: %v.", err)
	}
	for _, l := range links {
		if l.Attrs().Name == index {
			link = l
		}
	}
	qdiscs, err = netlink.QdiscList(link)
	if err != nil {
		logrus.Fatalf("Failed to fetch qdiscs: %v.", err)
	}
	classes, err = netlink.ClassList(link, netlink.HANDLE_NONE)
	if err != nil {
		logrus.Fatalf("Failed to fetch classes: %v.", err)
	}
	return
}

func FilterClass(classes *[]netlink.Class, filt string) (ret []netlink.Class) {
	for _, cl := range *classes {
		if cl.Type() == filt {
			ret = append(ret, cl)
		}
	}
	return
}

// CalcRates calculates the rates base don a maximum internet speed
func CalcRates(maxRate float64) (r rates) {
	ethRate := 1e9
	qosRatioC2 := 0.95
	qosRatioC11 := 0.4
	qosRatioC12 := 0.4
	qosRatioC13 := 0.2
	qosRatioC21 := 0.70
	qosRatioC31 := 0.7
	qosRatioC32 := 0.3
	qosRatioC22 := 0.10
	qosRatioC23 := 0.20
	qosRatioC3 := 0.2
	// Start calculating
	rateC2 := math.Ceil(maxRate * qosRatioC2)
	rateC13 := math.Ceil(rateC2 * qosRatioC13)
	rateC21 := math.Ceil(rateC13 * qosRatioC21)
	r = rates{
		MaxRate: uint32(maxRate),
		RateC1:  uint32(ethRate),
		RateC2:  uint32(rateC2),
		RateC11: uint32(math.Ceil(rateC2 * qosRatioC11)),
		RateC12: uint32(math.Ceil(rateC2 * qosRatioC12)),
		RateC13: uint32(math.Ceil(rateC2 * qosRatioC13)),
		RateC21: uint32(math.Ceil(rateC21)),
		RateC31: uint32(math.Ceil(rateC21 * qosRatioC31)),
		RateC32: uint32(math.Ceil(rateC21 * qosRatioC32)),
		RateC22: uint32(math.Ceil(rateC13 * qosRatioC22)),
		RateC23: uint32(math.Ceil(rateC13 * qosRatioC23)),
		RateC3:  uint32(math.Ceil(ethRate * qosRatioC3)),
	}
	return
}
