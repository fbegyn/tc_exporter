package tccollector_test

import (
	"os"
	"testing"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/go-kit/log"
	"github.com/jsimonetti/rtnetlink"
	"github.com/mdlayher/promtest"
)

func TestQdiscCollector(t *testing.T) {

	tests := []struct {
		ns     string
		name   string
		linkid uint32
	}{
		{ns: "default", name: "dummydefault", linkid: 999},
		{ns: "testing01", name: "dummy01", linkid: 1000},
		{ns: "testing02", name: "dummy1000", linkid: 1001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup the netns for testing
			if tt.ns != "default" {
				shell(t, "ip", "netns", "add", tt.ns)
				defer shell(t, "ip", "netns", "del", tt.ns)
				f, err := os.Open("/var/run/netns/" + tt.ns)
				if err != nil {
					t.Fatalf("failed to open namespace file: %v", err)
				}
				defer f.Close()
			}

			rtnl, err := setupDummyInterface(t, tt.ns, tt.name, tt.linkid)
			if err != nil {
				t.Fatalf("could not setup %s interface for %s: %v", tt.name, tt.ns, err)
			}
			defer rtnl.Link.Delete(tt.linkid)
			defer rtnl.Close()

			interf, err := getLinkByName(tt.ns, tt.name)
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("could not get %s interface by name", tt.name)
			}

			test := make(map[string][]rtnetlink.LinkMessage)
			con, _ := tcexporter.GetNetlinkConn("default")
			links, _ := con.Link.List()
			test["default"] = links
			con, _ = tcexporter.GetNetlinkConn(tt.ns)
			links, _ = con.Link.List()
			test[tt.ns] = links

			var logger log.Logger
			logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
			logger = log.With(logger, "test", "collector")

			qc, err := tcexporter.NewQdiscCollector(test, logger)
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("failed to create qdisc collector for %s: %v", interf.Attributes.Name, err)
			}

			body := promtest.Collect(t, qc)

			if !promtest.Lint(t, body) {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Errorf("one or more promlint errors found")
			}
			t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
			rtnl.Link.Delete(tt.linkid)
		})
	}

}
