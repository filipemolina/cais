package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/compose-spec/compose-go/v2/types"
)

func strPtr(s string) *string { return &s }

// The card shows the service's name (the key under services:) rather than
// container_name:, which most services leave unset - the bug was an empty
// "Name:" label while the list beside it showed the name.
func TestBasicInfoShowsTheServiceName(t *testing.T) {
	out := ansi.Strip(BasicInfo(types.ServiceConfig{
		Name:          "plex",
		ContainerName: "", // the common case: container_name is not set
		Image:         "lscr.io/plex:latest",
	}, 40))

	if !strings.Contains(out, "Name: plex") {
		t.Errorf("card does not show the service name:\n%s", out)
	}
}

// container_name is a separate concept from the service name; the card shows
// the name either way, matching what the list beside it shows.
func TestBasicInfoShowsTheServiceNameEvenWhenContainerNameIsSet(t *testing.T) {
	out := ansi.Strip(BasicInfo(types.ServiceConfig{
		Name:          "plex",
		ContainerName: "my-plex",
		Image:         "lscr.io/plex:latest",
	}, 40))

	if !strings.Contains(out, "Name: plex") {
		t.Errorf("card shows container_name %q instead of the service name:\n%s", "my-plex", out)
	}
}

// PUID/PGID are optional env-var-derived fields. When neither is set the row
// is dropped entirely, so the card never carries empty "PUID:  PGID: " labels.
func TestBasicInfoOmitsPuidPgidWhenNeitherIsSet(t *testing.T) {
	out := ansi.Strip(BasicInfo(types.ServiceConfig{
		Name:  "web",
		Image: "nginx:alpine",
	}, 40))

	if strings.Contains(out, "PUID:") {
		t.Errorf("card shows a PUID label with no value:\n%s", out)
	}
	if strings.Contains(out, "PGID:") {
		t.Errorf("card shows a PGID label with no value:\n%s", out)
	}
}

// When both are set the row shows them, as the *arr homelab stack expects.
func TestBasicInfoShowsPuidPgidWhenBothAreSet(t *testing.T) {
	out := ansi.Strip(BasicInfo(types.ServiceConfig{
		Name:        "radarr",
		Image:       "lscr.io/linuxserver/radarr:latest",
		Environment: types.MappingWithEquals{"PUID": strPtr("1000"), "PGID": strPtr("1000")},
	}, 40))

	if !strings.Contains(out, "PUID: 1000") {
		t.Errorf("card does not show PUID:\n%s", out)
	}
	if !strings.Contains(out, "PGID: 1000") {
		t.Errorf("card does not show PGID:\n%s", out)
	}
}

// When only one is set, only that one appears - no empty partner label.
func TestBasicInfoShowsPuidAloneWithoutAnEmptyPgid(t *testing.T) {
	out := ansi.Strip(BasicInfo(types.ServiceConfig{
		Name:        "sonarr",
		Image:       "lscr.io/linuxserver/sonarr:latest",
		Environment: types.MappingWithEquals{"PUID": strPtr("1000")},
	}, 40))

	if !strings.Contains(out, "PUID: 1000") {
		t.Errorf("card does not show PUID:\n%s", out)
	}
	if strings.Contains(out, "PGID:") {
		t.Errorf("card shows an empty PGID alongside a set PUID:\n%s", out)
	}
}
