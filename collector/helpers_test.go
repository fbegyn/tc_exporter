package tccollector_test

import (
	"os/exec"
	"syscall"
	"testing"

	tcexporter "github.com/fbegyn/tc_exporter/collector"
	"github.com/jsimonetti/rtnetlink"
	"golang.org/x/sys/unix"
)

// SetupDummyInterface installs a temporary dummy interface
func setupDummyInterface(t *testing.T, ns, iface string, linkindex uint32) (*rtnetlink.Conn, error) {

	con, err := tcexporter.GetNetlinkConn(ns)
	if err != nil {
		return &rtnetlink.Conn{}, err
	}

	if err := con.Link.New(&rtnetlink.LinkMessage{
		Family: unix.AF_UNSPEC,
		Type:   unix.ARPHRD_NETROM,
		Index:  linkindex,
		Flags:  unix.IFF_UP,
		Change: unix.IFF_UP,
		Attributes: &rtnetlink.LinkAttributes{
			Name: iface,
			Info: &rtnetlink.LinkInfo{Kind: "dummy"},
		},
	}); err != nil {
		return con, err
	}

	return con, nil
}

// getLinkByName fetches an interface from a netns based on the name. This is needed because the
// net.getInterfaceByName does not work for other netns
func getLinkByName(ns, link string) (rtnetlink.LinkMessage, error) {
	con, err := tcexporter.GetNetlinkConn(ns)
	if err != nil {
		return rtnetlink.LinkMessage{}, err
	}
	links, err := con.Link.List()
	if err != nil {
		return rtnetlink.LinkMessage{}, err
	}
	for _, l := range links {
		if l.Attributes.Name == link {
			return l, nil
		}
	}
	return rtnetlink.LinkMessage{}, nil
}

// thanks to mdlayher for this testing helper function
// https://github.com/mdlayher/netlink/blob/c558cf25207e57bc9cc026d2dd69e2ea2f6abd0e/conn_linux_integration_test.go#L617
func shell(t *testing.T, name string, arg ...string) {
	t.Helper()

	t.Logf("$ %s %v", name, arg)

	cmd := exec.Command(name, arg...)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start command %q: %v", name, err)
	}

	if err := cmd.Wait(); err != nil {
		// TODO(mdlayher): switch back to cmd.ProcessState.ExitCode() when we
		// drop support for Go 1.11.x.
		// Shell operations in these tests require elevated privileges.
		if cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus() == int(unix.EPERM) {
			t.Skipf("skipping, permission denied: %v", err)
		}

		t.Fatalf("failed to wait for command %q: %v", name, err)
	}
}
