package tccollector_test

import (
	"log/slog"
	"os"
	"strings"
	"testing"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/florianl/go-tc"
	"github.com/jsimonetti/rtnetlink"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

type cakeCollectorAdapter struct {
	col    tcexporter.ObjectCollector
	interf rtnetlink.LinkMessage
	qd     tc.Object
}

func (a cakeCollectorAdapter) Describe(ch chan<- *prometheus.Desc) { a.col.Describe(ch) }
func (a cakeCollectorAdapter) Collect(ch chan<- prometheus.Metric) {
	a.col.CollectObject(ch, "host", "default", a.interf, a.qd)
}

func TestCakeCollector(t *testing.T) {
	u32 := func(v uint32) *uint32 { return &v }
	u64 := func(v uint64) *uint64 { return &v }

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	col, err := tcexporter.NewCakeCollector(nil, logger)
	if err != nil {
		t.Fatalf("failed to create cake collector: %v", err)
	}

	qd := tc.Object{
		Msg: tc.Msg{Handle: 0x80010000, Parent: 0xffffffff},
		Attribute: tc.Attribute{
			Kind: "cake",
			XStats: &tc.XStats{Cake: &tc.CakeXStats{
				CapacityEstimate: u64(12500000),
				MemoryUsed:       u32(4096),
				Tins: []tc.CakeTinStats{
					{SentPackets: u32(10), SentBytes: u64(1500)},
					{SentPackets: u32(20)},
				},
			}},
		},
	}
	interf := rtnetlink.LinkMessage{Index: 3, Attributes: &rtnetlink.LinkAttributes{Name: "eth0"}}

	expected := `
# HELP tc_cake_capacity_estimate cake capacity estimate xstat in bytes per second
# TYPE tc_cake_capacity_estimate gauge
tc_cake_capacity_estimate{handle="8001:0",host="host",link="eth0",linkindex="3",netns="default",parent="ffff:ffff",type="cake"} 1.25e+07
# HELP tc_cake_memory_used cake memory used xstat
# TYPE tc_cake_memory_used gauge
tc_cake_memory_used{handle="8001:0",host="host",link="eth0",linkindex="3",netns="default",parent="ffff:ffff",type="cake"} 4096
# HELP tc_cake_tin_sent_bytes cake tin sent bytes xstat
# TYPE tc_cake_tin_sent_bytes counter
tc_cake_tin_sent_bytes{handle="8001:0",host="host",link="eth0",linkindex="3",netns="default",parent="ffff:ffff",tin="0",type="cake"} 1500
# HELP tc_cake_tin_sent_packets cake tin sent packets xstat
# TYPE tc_cake_tin_sent_packets counter
tc_cake_tin_sent_packets{handle="8001:0",host="host",link="eth0",linkindex="3",netns="default",parent="ffff:ffff",tin="0",type="cake"} 10
tc_cake_tin_sent_packets{handle="8001:0",host="host",link="eth0",linkindex="3",netns="default",parent="ffff:ffff",tin="1",type="cake"} 20
`
	adapter := cakeCollectorAdapter{col: col, interf: interf, qd: qd}
	if err := testutil.CollectAndCompare(adapter, strings.NewReader(expected)); err != nil {
		t.Errorf("unexpected cake metrics:\n%v", err)
	}
}
