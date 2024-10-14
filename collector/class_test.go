package tccollector_test

import (
	"log/slog"
	"os"
	"testing"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/jsimonetti/rtnetlink"
)

// TestClassCollector tests out the creation and polling of a ClassCollector
func TestClassCollector(t *testing.T) {
	// Define the test case
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
				// later on, we delete the netns since it is no longer usefull
				defer shell(t, "ip", "netns", "del", tt.ns)

				// Check if the namespace was actually created
				f, err := os.Open("/var/run/netns/" + tt.ns)
				if err != nil {
					t.Fatalf("failed to open namespace file: %v", err)
				}
				// Close the fd if we no longer need it
				defer f.Close()
			}

			// create the dummy interface in the netns
			rtnl, err := setupDummyInterface(t, tt.ns, tt.name, tt.linkid)
			if err != nil {
				t.Fatalf("could not setup %s interface for %s: %v", tt.name, tt.ns, err)
			}
			// close the returned rtnetlink connection if no longer needed
			defer rtnl.Close()

			// Fetch the test interface from the netns
			interf, err := getLinkByName(tt.ns, tt.name)
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("could not get %s interface by name", tt.name)
			}

			// Compose the test interface and netns "config"
			test := make(map[string][]rtnetlink.LinkMessage)
			// We fetcha ll devices in the default netns of the OS
			con, _ := tcexporter.GetNetlinkConn("default")
			links, _ := con.Link.List()
			test["default"] = links
			// We fetch all devices in the newly created netns
			con, _ = tcexporter.GetNetlinkConn(tt.ns)
			links, _ = con.Link.List()
			test[tt.ns] = links

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			logger = logger.With("test", "class")

			// Create a ClassCollector with the test "config"
			qc, err := tcexporter.NewClassCollector(test, logger)
			_ = qc
			if err != nil {
				t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
				rtnl.Link.Delete(tt.linkid)
				t.Fatalf("failed to create class collector for %s: %v", interf.Attributes.Name, err)
			}

			// // Test out the functionality of the collector
			// body := promtest.Collect(t, qc)
			// if !promtest.Lint(t, body) {
			// 	t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
			// 	rtnl.Link.Delete(tt.linkid)
			// 	t.Errorf("one or more promlint errors found")
			// }
			t.Logf("removing interface %s from %s\n", tt.name, tt.ns)
			rtnl.Link.Delete(tt.linkid)
		})
	}

}
