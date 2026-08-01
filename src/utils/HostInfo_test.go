package utils

import (
	"strings"
	"testing"
)

// Zorin (ID=zorin, ID_LIKE="ubuntu debian") is the regression test for the
// whole ID_LIKE decision: a family table keyed on ID alone has no entry for
// zorin (or pop, linuxmint, raspbian, endeavouros...) and would fall through
// to the generic message on the machine this app is developed on - see
// docs/plans/docker-preflight.md's Research section.
func TestParseOSReleaseZorinReachesDebianViaIDLike(t *testing.T) {
	id, idLike := parseOSRelease("ID=zorin\nID_LIKE=\"ubuntu debian\"\n")

	if id != "zorin" {
		t.Errorf("id = %q, want zorin", id)
	}
	if hostFamily(HostInfo{GOOS: "linux", DistroID: id, DistroLike: idLike}) != familyDebian {
		t.Errorf("family = %q, want debian", hostFamily(HostInfo{GOOS: "linux", DistroID: id, DistroLike: idLike}))
	}
}

func TestParseOSReleaseUbuntu(t *testing.T) {
	id, idLike := parseOSRelease("ID=ubuntu\nID_LIKE=debian\n")

	if id != "ubuntu" {
		t.Errorf("id = %q, want ubuntu", id)
	}
	if !contains(idLike, "debian") {
		t.Errorf("idLike = %v, want to contain debian", idLike)
	}
	if got := hostFamily(HostInfo{GOOS: "linux", DistroID: id, DistroLike: idLike}); got != familyDebian {
		t.Errorf("family = %q, want debian", got)
	}
}

func TestParseOSReleaseFedoraNoIDLike(t *testing.T) {
	id, idLike := parseOSRelease("ID=fedora\n")

	if id != "fedora" || len(idLike) != 0 {
		t.Errorf("got id=%q idLike=%v, want fedora with no ID_LIKE", id, idLike)
	}
	if got := hostFamily(HostInfo{GOOS: "linux", DistroID: id, DistroLike: idLike}); got != familyFedora {
		t.Errorf("family = %q, want fedora", got)
	}
}

func TestParseOSReleaseArch(t *testing.T) {
	id, _ := parseOSRelease("ID=arch\n")

	if got := hostFamily(HostInfo{GOOS: "linux", DistroID: id}); got != familyArch {
		t.Errorf("family = %q, want arch", got)
	}
}

func TestParseOSReleaseEmptyOrMissingIsGeneric(t *testing.T) {
	id, idLike := parseOSRelease("")

	if id != "" || len(idLike) != 0 {
		t.Errorf("got id=%q idLike=%v, want both empty", id, idLike)
	}
	if got := hostFamily(HostInfo{GOOS: "linux"}); got != familyGeneric {
		t.Errorf("family = %q, want generic", got)
	}
}

// The first match in ID_LIKE wins - an unrecognised ID with a multi-entry
// ID_LIKE resolves on whichever entry the table recognises first.
func TestParseOSReleaseIDLikeFirstMatchWins(t *testing.T) {
	_, idLike := parseOSRelease(`ID_LIKE="rhel centos fedora"` + "\n")

	if got := hostFamily(HostInfo{GOOS: "linux", DistroLike: idLike}); got != familyFedora {
		t.Errorf("family = %q, want fedora", got)
	}
}

func TestHostFamilyDarwinIgnoresDistroFields(t *testing.T) {
	if got := hostFamily(HostInfo{GOOS: "darwin", DistroID: "arch"}); got != familyDarwin {
		t.Errorf("family = %q, want darwin regardless of DistroID", got)
	}
}

// RemedyFor must never propose piping anything into a shell, and must never
// tell the user to run get.docker.com's one-liner (D2) - a test that pins
// D2 against a future "helpful" edit adding the install script back.
func TestRemedyForNeverPipesOrGetDockerCom(t *testing.T) {
	states := []DockerState{DockerNotInstalled, DockerComposeMissing, DockerDaemonUnreachable, DockerPermissionDenied}
	families := []HostInfo{
		{GOOS: "linux", DistroID: "debian"},
		{GOOS: "linux", DistroID: "ubuntu"},
		{GOOS: "linux", DistroID: "fedora"},
		{GOOS: "linux", DistroID: "arch"},
		{GOOS: "linux", DistroID: "opensuse"},
		{GOOS: "linux", DistroID: "unknown-distro"},
		{GOOS: "darwin"},
	}

	for _, state := range states {
		for _, host := range families {
			remedy := RemedyFor(DockerStatus{State: state}, host)

			if remedy.Summary == "" {
				t.Errorf("state=%v host=%+v: empty Summary", state, host)
			}
			for _, step := range remedy.Steps {
				if strings.Contains(step, "|") {
					t.Errorf("state=%v host=%+v: step %q pipes into a shell", state, host, step)
				}
				if strings.Contains(step, "get.docker.com") {
					t.Errorf("state=%v host=%+v: step %q runs get.docker.com's installer", state, host, step)
				}
			}
		}
	}
}

func TestRemedyForComposeV1FoundIsNamedExplicitly(t *testing.T) {
	remedy := RemedyFor(DockerStatus{State: DockerComposeMissing, ComposeV1Found: true}, HostInfo{GOOS: "linux"})

	if !strings.Contains(remedy.Note, "V1") {
		t.Errorf("Note = %q, want it to call out docker-compose V1 explicitly", remedy.Note)
	}
}

func TestRemedyForDaemonUnreachableNamesTheEndpoint(t *testing.T) {
	remedy := RemedyFor(DockerStatus{State: DockerDaemonUnreachable, Endpoint: "unix:///tmp/nope.sock"}, HostInfo{GOOS: "linux"})

	if !strings.Contains(remedy.Note, "unix:///tmp/nope.sock") {
		t.Errorf("Note = %q, want it to name the dialled endpoint", remedy.Note)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
