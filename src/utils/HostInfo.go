package utils

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

// HostInfo is enough about the machine to pick a remediation family: the OS,
// and on Linux, the distro's own idea of what it is and what it resembles.
type HostInfo struct {
	GOOS       string
	DistroID   string
	DistroLike []string
}

// DetectHost reads runtime.GOOS and, on Linux, /etc/os-release. A missing or
// unreadable file is not an error - it yields the generic family, same as
// any other unrecognised distro.
func DetectHost() HostInfo {
	host := HostInfo{GOOS: runtime.GOOS}
	if runtime.GOOS != "linux" {
		return host
	}

	contents, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return host
	}
	host.DistroID, host.DistroLike = parseOSRelease(string(contents))
	return host
}

// parseOSRelease extracts ID and ID_LIKE from /etc/os-release's contents:
// split on '=', strip quotes, keep the two keys this app needs.
func parseOSRelease(contents string) (id string, idLike []string) {
	for line := range strings.Lines(contents) {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"'`)

		switch key {
		case "ID":
			id = value
		case "ID_LIKE":
			idLike = strings.Fields(value)
		}
	}
	return id, idLike
}

// The platform families D7 covers, and nothing more: enough to pick a
// package manager, no further.
const (
	familyDebian  = "debian"
	familyFedora  = "fedora"
	familyArch    = "arch"
	familySuse    = "suse"
	familyDarwin  = "darwin"
	familyGeneric = "generic"
)

// linuxFamilies maps an /etc/os-release ID (or ID_LIKE entry) to the family
// whose package manager it shares. ID_LIKE reaches most of desktop Linux
// through the debian and fedora entries - see the Zorin case below.
var linuxFamilies = map[string]string{
	"debian": familyDebian,
	"ubuntu": familyDebian,

	"fedora": familyFedora,
	"rhel":   familyFedora,
	"centos": familyFedora,

	"arch": familyArch,

	"opensuse": familySuse,
	"suse":     familySuse,
}

// hostFamily picks the remediation family for host: match ID first, then
// each entry of ID_LIKE in order, then fall back to generic. A distro that
// matches nothing (there is no entry for zorin, pop, linuxmint, raspbian,
// endeavouros, or half of desktop Linux) still reaches its real family
// through ID_LIKE - see docs/plans/docker-preflight.md's Zorin case, which
// is why this walks ID_LIKE at all rather than keying on ID alone.
func hostFamily(host HostInfo) string {
	if host.GOOS == "darwin" {
		return familyDarwin
	}

	if family, ok := linuxFamilies[host.DistroID]; ok {
		return family
	}
	for _, like := range host.DistroLike {
		if family, ok := linuxFamilies[like]; ok {
			return family
		}
	}
	return familyGeneric
}

// Remedy is what the user should do about a DockerStatus on this machine:
// one sentence of what is wrong, and the exact command that fixes it - see
// D2 in docs/plans/docker-preflight.md for why Steps is printed, never run.
type Remedy struct {
	Summary string   // "The Docker daemon is not running."
	Steps   []string // shell lines, printed verbatim, never executed
	Note    string   // optional: the Snap/context/V1/rootless caveats
	DocsURL string
}

const (
	dockerInstallDocsURL     = "https://docs.docker.com/engine/install/"
	dockerPostInstallDocsURL = "https://docs.docker.com/engine/install/linux-postinstall/"
	dockerDesktopDocsURL     = "https://docs.docker.com/desktop/"
	dockerRootlessDocsURL    = "https://docs.docker.com/engine/security/rootless/"
)

// RemedyFor returns what the user should do about status on host. See D7:
// the states the app can fix in one line - 2, 3 and 4 - get a command; state
// 1 gets one only where the distro's own package manager carries docker
// directly (fedora, arch, suse), because the debian family's real answer is
// Docker's own multi-step repo setup, which this app does not embed (D2) -
// so it links out instead, same as the generic family.
func RemedyFor(status DockerStatus, host HostInfo) Remedy {
	family := hostFamily(host)

	switch status.State {
	case DockerNotInstalled:
		return notInstalledRemedy(family, status)
	case DockerComposeMissing:
		return composeMissingRemedy(family, status)
	case DockerDaemonUnreachable:
		return daemonUnreachableRemedy(host, status)
	case DockerPermissionDenied:
		return permissionDeniedRemedy(status)
	default:
		return Remedy{}
	}
}

func notInstalledRemedy(family string, status DockerStatus) Remedy {
	if status.Raw != "" {
		// isExecErrPermission's note: docker is on PATH but its execute bit
		// is stripped. There is no sixth state for this - see the edge case
		// in docs/plans/docker-preflight.md - so it is folded into state 1
		// with the detail kept rather than silently discarded.
		return Remedy{
			Summary: "Docker was found on PATH but is not executable.",
			Note:    status.Raw,
			DocsURL: dockerInstallDocsURL,
		}
	}

	switch family {
	case familyFedora:
		return Remedy{
			Summary: "Docker is not installed.",
			Steps:   []string{"sudo dnf install docker-ce docker-ce-cli containerd.io docker-compose-plugin"},
			DocsURL: dockerInstallDocsURL,
		}
	case familyArch:
		return Remedy{
			Summary: "Docker is not installed.",
			Steps:   []string{"sudo pacman -S docker docker-compose"},
			DocsURL: dockerInstallDocsURL,
		}
	case familySuse:
		return Remedy{
			Summary: "Docker is not installed.",
			Steps:   []string{"sudo zypper install docker docker-compose"},
			DocsURL: dockerInstallDocsURL,
		}
	case familyDarwin:
		return Remedy{
			Summary: "Docker is not installed.",
			Note:    "Install Docker Desktop, or run `brew install colima docker docker-compose` followed by `colima start`.",
			DocsURL: dockerDesktopDocsURL,
		}
	default:
		// familyDebian included deliberately: apt needs Docker's repo added
		// first (GPG key, source list, six commands that change over time),
		// and getting that wrong is worse than not printing it - D2. Link
		// out instead, same as anything unrecognised.
		return Remedy{
			Summary: "Docker is not installed.",
			DocsURL: dockerInstallDocsURL,
		}
	}
}

func composeMissingRemedy(family string, status DockerStatus) Remedy {
	remedy := Remedy{
		Summary: "The Docker Engine is installed, but the Compose v2 plugin is missing.",
		DocsURL: dockerInstallDocsURL,
	}

	switch family {
	case familyDebian:
		remedy.Steps = []string{"sudo apt-get install docker-compose-plugin"}
	case familyFedora:
		remedy.Steps = []string{"sudo dnf install docker-compose-plugin"}
	case familyArch:
		remedy.Steps = []string{"sudo pacman -S docker-compose"}
	case familySuse:
		remedy.Steps = []string{"sudo zypper install docker-compose"}
	case familyDarwin:
		remedy.Note = "Docker Desktop and colima both ship the Compose plugin already - if it is missing, the engine install itself needs attention."
	}

	if status.ComposeV1Found {
		// D6: docker-compose (V1) and docker compose (V2) look like the same
		// tool and are not. The app's container-status path depends on
		// `--format json`, which V1 never had.
		remedy.Note = "`docker-compose` (V1) is installed, but Cais needs the V2 plugin " +
			"(`docker compose`, no hyphen). V1 reached end of life in July 2023 and does not support `--format json`."
	}

	return remedy
}

func daemonUnreachableRemedy(host HostInfo, status DockerStatus) Remedy {
	remedy := Remedy{
		Summary: "The Docker daemon is not running.",
		DocsURL: dockerRootlessDocsURL,
	}

	if host.GOOS == "darwin" {
		remedy.Steps = []string{"Start Docker Desktop, or run: colima start"}
	} else {
		remedy.Steps = []string{"sudo systemctl start docker", "sudo systemctl enable --now docker"}
	}

	remedy.Note = endpointNote(status.Endpoint) +
		" Rootless docker needs no group membership and looks like this state when its own daemon is not running."
	return remedy
}

func permissionDeniedRemedy(status DockerStatus) Remedy {
	return Remedy{
		Summary: "The current user cannot reach the Docker socket - a permissions problem, not an outage.",
		Steps:   []string{"sudo usermod -aG docker $USER", "newgrp docker"},
		Note: endpointNote(status.Endpoint) +
			" Group membership is picked up at next login; `newgrp docker` is the immediate workaround for the " +
			"current shell. That group is root-equivalent - rootless mode is the alternative.",
		DocsURL: dockerPostInstallDocsURL,
	}
}

func endpointNote(endpoint string) string {
	if endpoint == "" {
		return ""
	}
	return fmt.Sprintf("Dialling %s - a leftover context from an uninstalled Docker Desktop looks exactly like this state with a perfectly healthy daemon underneath (`docker context show`).", endpoint)
}
