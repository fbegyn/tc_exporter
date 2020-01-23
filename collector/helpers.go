package collector

import (
	netlink "github.com/fbegyn/netlink-vishv"
	"github.com/sirupsen/logrus"
)

type NetlinkData struct {
	link    netlink.Link
	qdiscs  *[]netlink.Qdisc
	classes *[]netlink.Class
}

// GetData returns the tc data for the selected link
func GetData(link netlink.Link) (data NetlinkData) {
	// Get all the links from the system and select the one we need
	data.link = link

	// Fetch all qdiscs that are a part of the link
	qdiscs, err := netlink.QdiscList(data.link)
	if err != nil {
		logrus.Fatalf("Failed to fetch qdiscs: %v.", err)
	}
	data.qdiscs = &qdiscs

	// Fetch all the qdiscs that are a part of the link
	classes, err := netlink.ClassList(data.link, netlink.HANDLE_NONE)
	if err != nil {
		logrus.Fatalf("Failed to fetch classes: %v.", err)
	}
	data.classes = &classes
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
