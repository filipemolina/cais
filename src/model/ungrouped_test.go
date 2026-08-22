package model

import (
	"slices"
	"testing"

	"github.com/compose-spec/compose-go/v2/types"
	"github.com/filipemolina/cais/src/apptypes"
	"github.com/filipemolina/cais/src/utils"
)

// modelWithServices builds an AppModel holding the given services, the shape
// the helper tests below need.
func modelWithServices(services types.Services) AppModel {
	m := GetInitialModel(utils.ComposeSource{})
	m.config.configProject = &types.Project{Services: services}
	return m
}

// ungroupedServices is the derived membership of the reserved row: every
// service with no profiles: key, sorted.
func TestUngroupedServicesReturnsOnlyUntaggedServices(t *testing.T) {
	m := modelWithServices(types.Services{
		"web":   types.ServiceConfig{Name: "web", Profiles: []string{"core"}},
		"db":    types.ServiceConfig{Name: "db", Profiles: []string{"core"}},
		"proxy": types.ServiceConfig{Name: "proxy"},
		"cache": types.ServiceConfig{Name: "cache"},
	})

	got := m.ungroupedServices()
	want := []string{"cache", "proxy"}
	if !slices.Equal(got, want) {
		t.Errorf("ungroupedServices() = %v, want %v", got, want)
	}
}

// listedGroupNames appends the reserved row last when untagged services
// exist, so the groups list shows it after the real groups.
func TestListedGroupNamesAppendsUngroupedLast(t *testing.T) {
	m := modelWithServices(types.Services{
		"web":   types.ServiceConfig{Name: "web", Profiles: []string{"core"}},
		"proxy": types.ServiceConfig{Name: "proxy"},
	})

	got := m.listedGroupNames()
	want := []string{"core", apptypes.UngroupedGroup}
	if !slices.Equal(got, want) {
		t.Errorf("listedGroupNames() = %v, want %v", got, want)
	}
}

// When every service carries a profile there is nothing to derive the row
// from, so it must not appear.
func TestListedGroupNamesOmitsUngroupedWhenAllServicesAreTagged(t *testing.T) {
	m := modelWithServices(types.Services{
		"web": types.ServiceConfig{Name: "web", Profiles: []string{"core"}},
		"db":  types.ServiceConfig{Name: "db", Profiles: []string{"core"}},
	})

	got := m.listedGroupNames()
	want := []string{"core"}
	if !slices.Equal(got, want) {
		t.Errorf("listedGroupNames() = %v, want %v", got, want)
	}
}

// A hand-written real profile named ungrouped wins: the row must not be
// duplicated, and the real group is the only one shown.
func TestListedGroupNamesDoesNotDuplicateARealUngroupedProfile(t *testing.T) {
	m := modelWithServices(types.Services{
		"web":   types.ServiceConfig{Name: "web", Profiles: []string{apptypes.UngroupedGroup}},
		"proxy": types.ServiceConfig{Name: "proxy"},
	})

	got := m.listedGroupNames()
	want := []string{apptypes.UngroupedGroup}
	if !slices.Equal(got, want) {
		t.Errorf("listedGroupNames() = %v, want %v", got, want)
	}
}

// membersOf derives the ungrouped membership when no real profile carries
// the name, and falls back to the real profile's members when one does.
func TestMembersOfDerivesTheUngroupedMembership(t *testing.T) {
	m := modelWithServices(types.Services{
		"web":   types.ServiceConfig{Name: "web", Profiles: []string{"core"}},
		"proxy": types.ServiceConfig{Name: "proxy"},
		"cache": types.ServiceConfig{Name: "cache"},
	})

	got := m.membersOf(apptypes.UngroupedGroup)
	want := []string{"cache", "proxy"}
	if !slices.Equal(got, want) {
		t.Errorf("membersOf(ungrouped) = %v, want %v", got, want)
	}
}

func TestMembersOfUsesTheRealUngroupedProfileWhenOneExists(t *testing.T) {
	m := modelWithServices(types.Services{
		"web":   types.ServiceConfig{Name: "web", Profiles: []string{apptypes.UngroupedGroup}},
		"proxy": types.ServiceConfig{Name: "proxy"},
	})

	got := m.membersOf(apptypes.UngroupedGroup)
	want := []string{"web"}
	if !slices.Equal(got, want) {
		t.Errorf("membersOf(ungrouped) = %v, want %v", got, want)
	}
}

// homeStats counts real groups only: the derived row must not inflate the
// groups count, or the stats footer's ungrouped note would appear on a file
// with no real groups at all. This is the one that will silently rot if
// someone "tidies" allGroupNames later.
func TestHomeStatsGroupsCountIgnoresTheDerivedRow(t *testing.T) {
	m := modelWithServices(types.Services{
		"web":   types.ServiceConfig{Name: "web", Profiles: []string{"core"}},
		"proxy": types.ServiceConfig{Name: "proxy"},
	})

	groups, services, _, ungrouped := m.homeStats()

	if groups != 1 {
		t.Errorf("groups = %d, want 1 (the derived row must not count)", groups)
	}
	if services != 2 {
		t.Errorf("services = %d, want 2", services)
	}
	if ungrouped != 1 {
		t.Errorf("ungrouped = %d, want 1", ungrouped)
	}
}
