package collector

import (
	"os"

	netlink "github.com/fbegyn/netlink-vishv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

type linkCollector struct {
	link       netlink.Link
	operStatus *prometheus.Desc
}

func NewLinkCollector(link netlink.Link) (Collector, error) {
	module := "link"
	return &linkCollector{
		link: link,
		operStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, module, "operstatus"),
			"Operational status of the link from IFLA_OPERSTATE numeric representation of RFC2863",
			[]string{"host", "link", "type", "hwaddr"}, nil,
		),
	}, nil
}

func (c *linkCollector) Update(ch chan<- prometheus.Metric) error {
	host, err := os.Hostname()
	if err != nil {
		logrus.Errorf("couldn't get host name: %v\n", err)
		return err
	}

	ch <- prometheus.MustNewConstMetric(
		c.operStatus,
		prometheus.GaugeValue,
		float64(c.link.Attrs().OperState),
		host,
		c.link.Attrs().Name,
		c.link.Type(),
		c.link.Attrs().HardwareAddr.String(),
	)
	return nil
}
