package main

import (
	netlink "github.com/fbegyn/netlink-vishv"
	"github.com/sirupsen/logrus"
)

// GetData returns the tc data for the selected link
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

// FilterClass filters out unwanted entries from the list of classes
func FilterClass(classes *[]netlink.Class, filt string) (ret []netlink.Class) {
	for _, cl := range *classes {
		if cl.Type() == filt {
			ret = append(ret, cl)
		}
	}
	return
}
