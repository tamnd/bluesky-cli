package bluesky

import (
	"testing"

	"github.com/tamnd/any-cli/kit"
)

// These tests are offline: they exercise the domain metadata and operation
// wiring without making any network requests.

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "bluesky" {
		t.Errorf("Scheme = %q, want bluesky", info.Scheme)
	}
	if info.Identity.Binary != "bluesky" {
		t.Errorf("Identity.Binary = %q, want bluesky", info.Identity.Binary)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
}

func TestDomainOps(t *testing.T) {
	// All nine operations must be registered.
	want := []string{
		"feeds", "followers", "following", "profile",
		"search", "starter-packs", "thread", "trending", "user",
	}
	d := Domain{}
	info := d.Info()
	app := kit.New(info.Identity)
	d.Register(app)

	ops := app.Ops()
	got := map[string]bool{}
	for _, op := range ops {
		got[op.Meta().Name] = true
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("operation %q not registered", name)
		}
	}
	if len(ops) != len(want) {
		t.Errorf("got %d ops, want %d", len(ops), len(want))
	}
}

func TestDomainRegistered(t *testing.T) {
	// init() in domain.go registers the domain into the global kit registry.
	_, ok := kit.Lookup("bluesky")
	if !ok {
		t.Fatal("bluesky domain not found in kit registry after init()")
	}
}
