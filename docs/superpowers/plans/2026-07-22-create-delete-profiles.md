# Create/Delete Profiles Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let the user create a new profile (name a group + pick services to tag) and delete an existing one (untag it from every service), from the Home page's Groups panel, persisting both changes to the on-disk compose file.

**Architecture:** Introduce a single-modal-at-a-time overlay on `AppModel` (`activeModal tea.Model`), rendered as a real compositor layer on top of the existing screen and given exclusive control of key input while open. Two new leaf modal components (`ConfirmModal`, and a two-step `ProfileNameModal` → `ServiceChecklistModal` chain) collect input and hand off a `tea.Cmd` to run once they close. That cmd calls into a new pure `src/utils/ProfileTags.go`, which edits the compose file's already-parsed `yaml.Node` tree in place (so comments/formatting survive) and writes it back. A successful write re-triggers the existing `cmds.GetConfig` → `configSyncCmds()` refresh path, so there's no separate in-memory profile state to keep in sync — disk stays the source of truth, exactly as it is today.

**Tech Stack:** Go 1.26, Bubble Tea v2 / Bubbles v2 / Lip Gloss v2 (`charm.land/...`), `github.com/compose-spec/compose-go/v2`, `gopkg.in/yaml.v3` (already in the module graph transitively — this plan promotes it to a direct `go.mod` requirement).

## Global Constraints

- Module: `stack-stitcher`, Go 1.26.4 (`go.mod:1-3`).
- Follow the existing per-component `Init`/`Update`/`View` pattern; one exported constructor function per component file (see `src/components/ProfilesList.go`, `src/components/GroupDetailsPanel.go`).
- All cross-component communication goes through `src/cmds/` message types dispatched as `tea.Cmd` — never call between components directly.
- Errors that come from Docker/file-system actions surface via `AppModel.lastError` and the existing red banner in `src/model/View.go` (`errorBannerStyle`) — do not invent a second error-display mechanism.
- No new test framework: this repo has zero `*_test.go` files today. Only `src/utils/ProfileTags.go` gets tests in this plan (pure logic, file-corrupting risk if wrong); everything else (components, cmds, wiring) follows the existing untested convention and is checked with `go build ./...` plus the manual verification in Task 8.
- Design reference: `docs/superpowers/specs/2026-07-22-create-delete-profiles-design.md`.

---

### Task 1: Compose-file profile tag editing (`src/utils/ProfileTags.go`)

**Files:**
- Create: `src/utils/ProfileTags.go`
- Test: `src/utils/ProfileTags_test.go`
- Modify: `go.mod`, `go.sum` (via `go mod tidy`, not by hand)

**Interfaces:**
- Consumes: nothing project-specific — takes a compose file path directly.
- Produces:
  - `func AddProfileTag(fileName string, profileName string, serviceNames []string) error`
  - `func RemoveProfileTag(fileName string, profileName string) error`
  - Both are consumed by Task 2's `cmds.CreateProfile` / `cmds.DeleteProfile`.

- [ ] **Step 1: Write the failing tests**

Create `src/utils/ProfileTags_test.go`:

```go
package utils

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const baseFixture = `services:
  app:
    image: nginx:alpine
    profiles: ["core"] # core services
    ports:
      - "8085:80"

  db:
    image: postgres:alpine
    profiles: ["core"]

  cache:
    image: redis:alpine
`

const multiTagFixture = `services:
  app:
    image: nginx:alpine
    profiles: ["core", "extra"]

  db:
    image: postgres:alpine
    profiles: ["core"]
`

func writeFixture(t *testing.T, contents string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	return path
}

func readServiceProfiles(t *testing.T, path, service string) []string {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	var doc struct {
		Services map[string]struct {
			Profiles []string `yaml:"profiles"`
		} `yaml:"services"`
	}

	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parsing result file: %v", err)
	}

	return doc.Services[service].Profiles
}

func hasProfilesKey(t *testing.T, path, service string) bool {
	t.Helper()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	var doc map[string]map[string]map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parsing result file: %v", err)
	}

	_, ok := doc["services"][service]["profiles"]
	return ok
}

func TestAddProfileTag_NoExistingKey(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := AddProfileTag(path, "extra", []string{"cache"}); err != nil {
		t.Fatalf("AddProfileTag: %v", err)
	}

	got := readServiceProfiles(t, path, "cache")
	want := []string{"extra"}

	if !slices.Equal(got, want) {
		t.Errorf("cache profiles = %v, want %v", got, want)
	}
}

func TestAddProfileTag_ExistingKeyPreservesComment(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := AddProfileTag(path, "extra", []string{"app"}); err != nil {
		t.Fatalf("AddProfileTag: %v", err)
	}

	got := readServiceProfiles(t, path, "app")
	want := []string{"core", "extra"}

	if !slices.Equal(got, want) {
		t.Errorf("app profiles = %v, want %v", got, want)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	if !strings.Contains(string(raw), "core services") {
		t.Errorf("expected line comment to survive the edit, got:\n%s", raw)
	}
}

func TestAddProfileTag_Idempotent(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := AddProfileTag(path, "core", []string{"app"}); err != nil {
		t.Fatalf("AddProfileTag: %v", err)
	}

	got := readServiceProfiles(t, path, "app")
	want := []string{"core"}

	if !slices.Equal(got, want) {
		t.Errorf("app profiles = %v, want %v (should not duplicate)", got, want)
	}
}

func TestRemoveProfileTag_LastTagDropsKey(t *testing.T) {
	path := writeFixture(t, baseFixture)

	if err := RemoveProfileTag(path, "core"); err != nil {
		t.Fatalf("RemoveProfileTag: %v", err)
	}

	if hasProfilesKey(t, path, "app") {
		t.Errorf("expected app's profiles key to be removed entirely")
	}

	if hasProfilesKey(t, path, "db") {
		t.Errorf("expected db's profiles key to be removed entirely")
	}
}

func TestRemoveProfileTag_OneOfSeveral(t *testing.T) {
	path := writeFixture(t, multiTagFixture)

	if err := RemoveProfileTag(path, "extra"); err != nil {
		t.Fatalf("RemoveProfileTag: %v", err)
	}

	got := readServiceProfiles(t, path, "app")
	want := []string{"core"}

	if !slices.Equal(got, want) {
		t.Errorf("app profiles = %v, want %v", got, want)
	}

	if !hasProfilesKey(t, path, "app") {
		t.Errorf("expected app's profiles key to survive (still has %q)", "core")
	}

	dbGot := readServiceProfiles(t, path, "db")
	if !slices.Equal(dbGot, []string{"core"}) {
		t.Errorf("db profiles = %v, want unchanged [core]", dbGot)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail (function not defined)**

Run: `go test ./src/utils/... -run TestAddProfileTag -v`
Expected: FAIL — `undefined: AddProfileTag` (and similarly for `RemoveProfileTag`)

- [ ] **Step 3: Write the implementation**

Create `src/utils/ProfileTags.go`:

```go
package utils

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// AddProfileTag tags each of the given services with profileName in the
// compose file at fileName, preserving the file's existing formatting and
// comments as much as possible. It's idempotent: a service that already
// carries the tag is left unchanged.
func AddProfileTag(fileName string, profileName string, serviceNames []string) error {
	doc, err := readComposeNode(fileName)
	if err != nil {
		return err
	}

	servicesNode, err := servicesMappingNode(doc)
	if err != nil {
		return err
	}

	for _, serviceName := range serviceNames {
		serviceNode := findMappingValue(servicesNode, serviceName)
		if serviceNode == nil {
			return fmt.Errorf("service %q not found in compose file", serviceName)
		}

		profilesNode := findMappingValue(serviceNode, "profiles")
		if profilesNode == nil {
			profilesNode = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
			serviceNode.Content = append(serviceNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: "profiles"},
				profilesNode,
			)
		}

		if !sequenceContains(profilesNode, profileName) {
			profilesNode.Content = append(profilesNode.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: profileName,
			})
		}
	}

	return writeComposeNode(fileName, doc)
}

// RemoveProfileTag strips profileName from every service in the compose
// file at fileName that carries it. A service's profiles key is removed
// entirely, rather than left as an empty list, once its last tag is gone.
func RemoveProfileTag(fileName string, profileName string) error {
	doc, err := readComposeNode(fileName)
	if err != nil {
		return err
	}

	servicesNode, err := servicesMappingNode(doc)
	if err != nil {
		return err
	}

	// Mapping content is a flat, alternating slice: Content[0] is a key,
	// Content[1] is its value, and so on.
	for i := 0; i+1 < len(servicesNode.Content); i += 2 {
		removeProfileFromService(servicesNode.Content[i+1], profileName)
	}

	return writeComposeNode(fileName, doc)
}

func removeProfileFromService(serviceNode *yaml.Node, profileName string) {
	for i := 0; i+1 < len(serviceNode.Content); i += 2 {
		if serviceNode.Content[i].Value != "profiles" {
			continue
		}

		profilesNode := serviceNode.Content[i+1]
		remaining := profilesNode.Content[:0]
		for _, item := range profilesNode.Content {
			if item.Value != profileName {
				remaining = append(remaining, item)
			}
		}
		profilesNode.Content = remaining

		if len(profilesNode.Content) == 0 {
			serviceNode.Content = append(serviceNode.Content[:i], serviceNode.Content[i+2:]...)
		}

		return
	}
}

func readComposeNode(fileName string) (*yaml.Node, error) {
	raw, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed reading %s: %w", fileName, err)
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("failed parsing %s: %w", fileName, err)
	}

	return &doc, nil
}

func writeComposeNode(fileName string, doc *yaml.Node) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	if err := enc.Encode(doc); err != nil {
		return fmt.Errorf("failed encoding %s: %w", fileName, err)
	}
	if err := enc.Close(); err != nil {
		return fmt.Errorf("failed encoding %s: %w", fileName, err)
	}

	if err := os.WriteFile(fileName, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed writing %s: %w", fileName, err)
	}

	return nil
}

func servicesMappingNode(doc *yaml.Node) (*yaml.Node, error) {
	if len(doc.Content) == 0 {
		return nil, fmt.Errorf("compose file is empty")
	}

	servicesNode := findMappingValue(doc.Content[0], "services")
	if servicesNode == nil {
		return nil, fmt.Errorf("compose file has no top-level services key")
	}

	return servicesNode, nil
}

// findMappingValue returns the value node for key in mapping, or nil if
// the key isn't present.
func findMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}

	return nil
}

func sequenceContains(sequence *yaml.Node, value string) bool {
	for _, item := range sequence.Content {
		if item.Value == value {
			return true
		}
	}

	return false
}
```

- [ ] **Step 4: Pull `gopkg.in/yaml.v3` into `go.mod` as a direct dependency**

Run: `go mod tidy`
Expected: `go.mod` gains a `gopkg.in/yaml.v3 v3.0.1` line in the first `require (...)` block (it's already in `go.sum` transitively, so no network fetch is needed).

- [ ] **Step 5: Run the tests to verify they pass**

Run: `go test ./src/utils/... -v`
Expected: PASS — all of `TestAddProfileTag_NoExistingKey`, `TestAddProfileTag_ExistingKeyPreservesComment`, `TestAddProfileTag_Idempotent`, `TestRemoveProfileTag_LastTagDropsKey`, `TestRemoveProfileTag_OneOfSeveral`

- [ ] **Step 6: Commit**

```bash
git add src/utils/ProfileTags.go src/utils/ProfileTags_test.go go.mod go.sum
git commit -m "Add compose-file profile tag editing with comment-preserving YAML edits"
```

---

### Task 2: `cmds` messages for profile create/delete and modal open/close

**Files:**
- Create: `src/cmds/CreateProfile.go`
- Create: `src/cmds/DeleteProfile.go`
- Create: `src/cmds/OpenCreateProfileModal.go`
- Create: `src/cmds/OpenDeleteProfileModal.go`
- Create: `src/cmds/CloseModal.go`

**Interfaces:**
- Consumes: `utils.GetComposeFileName() (string, error)` (`src/utils/GetComposeFileName.go`), `utils.AddProfileTag`/`utils.RemoveProfileTag` (Task 1).
- Produces (consumed by Task 6's `AppModel.Update` and Tasks 3-5's modals):
  - `type CreateProfileMsg struct { Err error }` / `func CreateProfile(name string, serviceNames []string) tea.Cmd`
  - `type DeleteProfileMsg struct { Err error }` / `func DeleteProfile(name string) tea.Cmd`
  - `type OpenCreateProfileModalMsg struct{}` / `func OpenCreateProfileModal() tea.Cmd`
  - `type OpenDeleteProfileModalMsg string` / `func OpenDeleteProfileModal(profileName string) tea.Cmd`
  - `type CloseModalMsg struct { Follow tea.Cmd }` / `func CloseModal(follow tea.Cmd) tea.Cmd`

- [ ] **Step 1: Create the profile action cmds**

`src/cmds/CreateProfile.go`:

```go
package cmds

import (
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type CreateProfileMsg struct {
	Err error
}

// CreateProfile tags each of the given services with a new profile name in
// the compose file on disk.
func CreateProfile(name string, serviceNames []string) tea.Cmd {
	return func() tea.Msg {
		fileName, err := utils.GetComposeFileName()
		if err != nil {
			return CreateProfileMsg{Err: err}
		}

		return CreateProfileMsg{Err: utils.AddProfileTag(fileName, name, serviceNames)}
	}
}
```

`src/cmds/DeleteProfile.go`:

```go
package cmds

import (
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type DeleteProfileMsg struct {
	Err error
}

// DeleteProfile removes a profile tag from every service that carries it
// in the compose file on disk.
func DeleteProfile(name string) tea.Cmd {
	return func() tea.Msg {
		fileName, err := utils.GetComposeFileName()
		if err != nil {
			return DeleteProfileMsg{Err: err}
		}

		return DeleteProfileMsg{Err: utils.RemoveProfileTag(fileName, name)}
	}
}
```

- [ ] **Step 2: Create the modal open/close cmds**

`src/cmds/OpenCreateProfileModal.go`:

```go
package cmds

import tea "charm.land/bubbletea/v2"

type OpenCreateProfileModalMsg struct{}

func OpenCreateProfileModal() tea.Cmd {
	return func() tea.Msg { return OpenCreateProfileModalMsg{} }
}
```

`src/cmds/OpenDeleteProfileModal.go`:

```go
package cmds

import tea "charm.land/bubbletea/v2"

type OpenDeleteProfileModalMsg string

func OpenDeleteProfileModal(profileName string) tea.Cmd {
	return func() tea.Msg { return OpenDeleteProfileModalMsg(profileName) }
}
```

`src/cmds/CloseModal.go`:

```go
package cmds

import tea "charm.land/bubbletea/v2"

// CloseModalMsg tells AppModel to clear the active modal. Follow, if set,
// is appended to the batch of commands run once the modal is gone - this is
// how a modal hands off the action it collected input for (e.g. actually
// creating a profile) without needing to know about AppModel itself.
type CloseModalMsg struct {
	Follow tea.Cmd
}

func CloseModal(follow tea.Cmd) tea.Cmd {
	return func() tea.Msg { return CloseModalMsg{Follow: follow} }
}
```

- [ ] **Step 3: Verify the package builds**

Run: `go build ./...`
Expected: no output, exit code 0

- [ ] **Step 4: Commit**

```bash
git add src/cmds/CreateProfile.go src/cmds/DeleteProfile.go src/cmds/OpenCreateProfileModal.go src/cmds/OpenDeleteProfileModal.go src/cmds/CloseModal.go
git commit -m "Add cmds for profile create/delete and modal open/close"
```

---

### Task 3: `ConfirmModal` component (delete confirmation)

**Files:**
- Create: `src/components/ConfirmModal.go`

**Interfaces:**
- Consumes: `cmds.CloseModal(follow tea.Cmd) tea.Cmd` (Task 2).
- Produces: `func ConfirmModal(message string, confirm tea.Cmd) tea.Model` — consumed by Task 6's `AppModel.Update` (`OpenDeleteProfileModalMsg` handler).

- [ ] **Step 1: Write the component**

`src/components/ConfirmModal.go`:

```go
package components

import (
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/cmds"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ConfirmModalModel struct {
	message string
	confirm tea.Cmd
}

func (m ConfirmModalModel) Init() tea.Cmd {
	return nil
}

func (m ConfirmModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "y":
		return m, cmds.CloseModal(m.confirm)
	case "n", "esc":
		return m, cmds.CloseModal(nil)
	}

	return m, nil
}

func (m ConfirmModalModel) View() tea.View {
	style := lipgloss.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.PrimaryColor).
		Background(appstyles.PanelBackgroundColor)

	return tea.NewView(style.Render(m.message))
}

// ConfirmModal shows message and, if the user presses 'y', runs confirm
// once the modal closes. 'n' or Esc dismisses without running it.
func ConfirmModal(message string, confirm tea.Cmd) tea.Model {
	return ConfirmModalModel{
		message: message,
		confirm: confirm,
	}
}
```

- [ ] **Step 2: Verify the package builds**

Run: `go build ./...`
Expected: no output, exit code 0

- [ ] **Step 3: Commit**

```bash
git add src/components/ConfirmModal.go
git commit -m "Add ConfirmModal component"
```

---

### Task 4: `ServiceChecklistModal` component (create-profile step 2)

**Files:**
- Create: `src/apptypes/CheckableServiceItem.go`
- Create: `src/components/ServiceChecklistModal.go`

**Interfaces:**
- Consumes: `cmds.CloseModal` and `cmds.CreateProfile` (Task 2).
- Produces:
  - `type apptypes.CheckableServiceItem struct { Name string; Checked bool }` (implements `list.Item`)
  - `func ServiceChecklistModal(profileName string, serviceNames []string) tea.Model` — consumed by Task 5's `ProfileNameModal` (as the Enter-key transition target).

- [ ] **Step 1: Write the list item type**

`src/apptypes/CheckableServiceItem.go`:

```go
package apptypes

import "fmt"

// CheckableServiceItem is a list.Item for the service-selection checklist
// shown when creating a profile.
type CheckableServiceItem struct {
	Name    string
	Checked bool
}

func (s CheckableServiceItem) Title() string {
	box := "[ ]"
	if s.Checked {
		box = "[x]"
	}

	return fmt.Sprintf("%s %s", box, s.Name)
}

func (s CheckableServiceItem) FilterValue() string { return s.Name }
```

- [ ] **Step 2: Write the modal component**

`src/components/ServiceChecklistModal.go`:

```go
package components

import (
	"fmt"
	"io"
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/apptypes"
	"stack-stitcher/src/cmds"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type serviceChecklistDelegate struct{}

func (d serviceChecklistDelegate) Height() int                            { return 1 }
func (d serviceChecklistDelegate) Spacing() int                           { return 0 }
func (d serviceChecklistDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d serviceChecklistDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	item, ok := listItem.(apptypes.CheckableServiceItem)
	if !ok {
		return
	}

	style := lipgloss.NewStyle().Foreground(appstyles.SecondaryFontColor)
	if index == m.Index() {
		style = style.Foreground(appstyles.PrimaryFontColor).Bold(true)
	}

	fmt.Fprint(w, style.Render(item.Title()))
}

type ServiceChecklistModalModel struct {
	profileName string
	list        list.Model
}

func (m ServiceChecklistModalModel) Init() tea.Cmd {
	return nil
}

func (m ServiceChecklistModalModel) checkedServiceNames() []string {
	var names []string

	for _, listItem := range m.list.Items() {
		if item, ok := listItem.(apptypes.CheckableServiceItem); ok && item.Checked {
			names = append(names, item.Name)
		}
	}

	return names
}

func (m ServiceChecklistModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var finalCmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "esc":
			return m, cmds.CloseModal(nil)

		case "space":
			index := m.list.GlobalIndex()
			if item, ok := m.list.SelectedItem().(apptypes.CheckableServiceItem); ok {
				item.Checked = !item.Checked
				finalCmds = append(finalCmds, m.list.SetItem(index, item))
			}

		case "enter":
			if checked := m.checkedServiceNames(); len(checked) > 0 {
				return m, cmds.CloseModal(cmds.CreateProfile(m.profileName, checked))
			}
		}
	}

	var listCmd tea.Cmd
	m.list, listCmd = m.list.Update(msg)
	finalCmds = append(finalCmds, listCmd)

	return m, tea.Batch(finalCmds...)
}

func (m ServiceChecklistModalModel) View() tea.View {
	style := lipgloss.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.PrimaryColor).
		Background(appstyles.PanelBackgroundColor)

	return tea.NewView(style.Render(m.list.View()))
}

// ServiceChecklistModal is step 2 of the create-profile flow: pick which
// services get tagged with profileName. Space toggles the highlighted
// service, Enter confirms (requires at least one checked), Esc cancels the
// whole create flow.
func ServiceChecklistModal(profileName string, serviceNames []string) tea.Model {
	items := make([]list.Item, 0, len(serviceNames))
	for _, name := range serviceNames {
		items = append(items, apptypes.CheckableServiceItem{Name: name})
	}

	checklist := list.New(items, serviceChecklistDelegate{}, 40, len(items)+2)
	checklist.Title = fmt.Sprintf("Select services for %q", profileName)
	checklist.SetShowHelp(false)
	checklist.SetShowStatusBar(false)
	checklist.Styles.Title = checklist.Styles.Title.Background(appstyles.PrimaryColor)

	return ServiceChecklistModalModel{
		profileName: profileName,
		list:        checklist,
	}
}
```

- [ ] **Step 3: Verify the package builds**

Run: `go build ./...`
Expected: no output, exit code 0

- [ ] **Step 4: Commit**

```bash
git add src/apptypes/CheckableServiceItem.go src/components/ServiceChecklistModal.go
git commit -m "Add ServiceChecklistModal component"
```

---

### Task 5: `ProfileNameModal` component (create-profile step 1)

**Files:**
- Create: `src/components/ProfileNameModal.go`

**Interfaces:**
- Consumes: `cmds.CloseModal` (Task 2), `components.ServiceChecklistModal(profileName string, serviceNames []string) tea.Model` (Task 4).
- Produces: `func ProfileNameModal(existingProfiles []string, serviceNames []string) tea.Model` — consumed by Task 6's `AppModel.Update` (`OpenCreateProfileModalMsg` handler).

- [ ] **Step 1: Write the component**

`src/components/ProfileNameModal.go`:

```go
package components

import (
	"fmt"
	"slices"
	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/cmds"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ProfileNameModalModel struct {
	input            textinput.Model
	existingProfiles []string
	serviceNames     []string
	errMsg           string
}

func (m ProfileNameModalModel) Init() tea.Cmd {
	return nil
}

func (m ProfileNameModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyMsg.String() {
		case "esc":
			return m, cmds.CloseModal(nil)

		case "enter":
			name := m.input.Value()

			if name == "" {
				m.errMsg = "Profile name can't be empty"
				return m, nil
			}

			if slices.Contains(m.existingProfiles, name) {
				m.errMsg = fmt.Sprintf("Profile %q already exists", name)
				return m, nil
			}

			return ServiceChecklistModal(name, m.serviceNames), nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)

	return m, cmd
}

func (m ProfileNameModalModel) View() tea.View {
	style := lipgloss.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.PrimaryColor).
		Background(appstyles.PanelBackgroundColor)

	lines := []string{"New profile name:", m.input.View()}
	if m.errMsg != "" {
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#B33A3A"))
		lines = append(lines, errStyle.Render(m.errMsg))
	}

	return tea.NewView(style.Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

// ProfileNameModal is step 1 of the create-profile flow: prompt for a new,
// unique profile name. Enter with a valid name advances to
// ServiceChecklistModal; Esc cancels the whole flow.
func ProfileNameModal(existingProfiles []string, serviceNames []string) tea.Model {
	input := textinput.New()
	input.Placeholder = "e.g. core"
	input.SetWidth(30)
	input.Focus()

	return ProfileNameModalModel{
		input:            input,
		existingProfiles: existingProfiles,
		serviceNames:     serviceNames,
	}
}
```

- [ ] **Step 2: Verify the package builds**

Run: `go build ./...`
Expected: no output, exit code 0

- [ ] **Step 3: Commit**

```bash
git add src/components/ProfileNameModal.go
git commit -m "Add ProfileNameModal component"
```

---

### Task 6: Wire the modal overlay into `AppModel`

**Files:**
- Modify: `src/model/AppModel.go`
- Modify: `src/model/Update.go`
- Modify: `src/model/View.go`

**Interfaces:**
- Consumes: `components.ConfirmModal`, `components.ProfileNameModal` (Tasks 3-5); `cmds.OpenCreateProfileModalMsg`, `cmds.OpenDeleteProfileModalMsg`, `cmds.CloseModalMsg`, `cmds.CreateProfileMsg`, `cmds.DeleteProfileMsg`, `cmds.DeleteProfile` (Task 2); `utils.Deduplicate` (existing, `src/utils/Deduplicate.go`).
- Produces: `AppModel.activeModal tea.Model` field and `func (m AppModel) allProfileNames() []string` — consumed by Task 7 indirectly (Task 7 only needs the `cmds.Open*` functions, already available from Task 2, but relies on this task's routing to actually open the modal).

- [ ] **Step 1: Add the `activeModal` field and `allProfileNames` helper to `AppModel.go`**

Modify `src/model/AppModel.go`. Change the import block from:

```go
import (
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/components"
	"stack-stitcher/src/constants"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)
```

to:

```go
import (
	"slices"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/components"
	"stack-stitcher/src/constants"
	"stack-stitcher/src/utils"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)
```

Change the `AppModel` struct from:

```go
type AppModel struct {
	navigation       navigationModel
	config           configModel
	containers       containersModel
	pages            map[string][]tea.Model
	activePage       string
	components       Components
	focusedComponent int
	lastError        string
}
```

to:

```go
type AppModel struct {
	navigation       navigationModel
	config           configModel
	containers       containersModel
	pages            map[string][]tea.Model
	activePage       string
	components       Components
	focusedComponent int
	lastError        string
	activeModal      tea.Model
}
```

Add this method right after `ChangeFocus` (before `UpdateInnerComponent`):

```go
// allProfileNames returns every distinct profile referenced by any service
// in the loaded compose project, sorted. Returns nil if no project is
// loaded yet.
func (m AppModel) allProfileNames() []string {
	if m.config.configProject == nil {
		return nil
	}

	var profiles []string
	for _, service := range m.config.configProject.Services {
		profiles = append(profiles, service.Profiles...)
	}

	profiles = utils.Deduplicate(profiles)
	slices.Sort(profiles)

	return profiles
}
```

- [ ] **Step 2: Route input to the modal and handle the new messages in `Update.go`**

Replace the full contents of `src/model/Update.go` with:

```go
package model

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/components"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)

// configSyncCmds re-derives the ordered services/profiles lists from the
// loaded compose project and broadcasts them. Messages only reach the
// currently active page's components (see UpdateInnerComponent), so this
// needs to run both right after the config loads AND whenever the active
// page changes - otherwise a page that wasn't active at load time (e.g.
// Dashboard, since Home is active first) would never receive its services.
func (m AppModel) configSyncCmds() []tea.Cmd {
	if m.config.configProject == nil {
		return nil
	}

	var syncCmds []tea.Cmd

	length := len(m.config.configProject.Services) + len(m.config.configProject.DisabledServices)
	orderedServices := make([]types.ServiceConfig, 0, length)

	orderedServicesMap := m.config.configProject.Services
	maps.Copy(orderedServicesMap, m.config.configProject.DisabledServices)

	for _, service := range orderedServicesMap {
		orderedServices = append(orderedServices, service)
	}

	slices.SortFunc(orderedServices, func(a, b types.ServiceConfig) int {
		return cmp.Compare(a.Name, b.Name)
	})

	syncCmds = append(syncCmds, cmds.SetServicesList(orderedServices))
	if len(orderedServices) > 0 {
		syncCmds = append(syncCmds, cmds.SetSelectedService(orderedServices[0]))
	}

	orderedProfiles := m.allProfileNames()

	syncCmds = append(syncCmds, cmds.SetProfilesList(orderedProfiles))
	if len(orderedProfiles) > 0 {
		syncCmds = append(syncCmds, cmds.SetSelectedProfile(orderedProfiles[0]))
	}

	return syncCmds
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// This var contains all the cmds that should be executed
	// at the end. Those can come from this model or from any of the
	// nested models in m.components
	var finalCmds []tea.Cmd

	// While a modal is open, it owns all key input exclusively - the
	// underlying panels and Tab/quit handling are frozen until it closes.
	if m.activeModal != nil {
		if _, ok := msg.(tea.KeyPressMsg); ok {
			var modalCmd tea.Cmd
			m.activeModal, modalCmd = m.activeModal.Update(msg)
			return m, modalCmd
		}
	}

	switch msg := msg.(type) {
	// Handle keyboard events
	case tea.KeyPressMsg:
		switch msg.String() {

		// Quit the program on Ctrl+c or q
		case "ctrl+c", "q":
			return m, tea.Quit

		case "tab":
			tabCmd := m.ChangeFocus(nil)
			finalCmds = append(finalCmds, tabCmd)

		case "shift+tab":
			idx := int(-1)
			tabCmd := m.ChangeFocus(&idx)
			finalCmds = append(finalCmds, tabCmd)
		}

	// This is executed once when the app loads and after every
	// window resize.
	case tea.WindowSizeMsg:
		m.config.terminalWidht = msg.Width
		m.config.terminalHeight = msg.Height

	// Commands from the cmds folder
	case cmds.SetActivePageMsg:
		m.activePage = string(msg)
		// Refresh container state, and re-sync services/profiles, so the
		// newly active page's components have data to show even if they
		// weren't active when it was first loaded.
		finalCmds = append(finalCmds, cmds.GetRunningContainers)
		finalCmds = append(finalCmds, m.configSyncCmds()...)

	case cmds.GetRunningContainersMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
		}

	case cmds.DockerActionMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetRunningContainers)
		}

	case cmds.GetConfigMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
			break
		}

		m.config.configFileName = msg.FileName
		m.config.configProject = msg.Project
		finalCmds = append(finalCmds, m.configSyncCmds()...)

	case cmds.OpenCreateProfileModalMsg:
		if m.config.configProject != nil {
			m.activeModal = components.ProfileNameModal(m.allProfileNames(), m.config.configProject.ServiceNames())
		}

	case cmds.OpenDeleteProfileModalMsg:
		profileName := string(msg)
		m.activeModal = components.ConfirmModal(
			fmt.Sprintf("Delete profile %q? (y/n)", profileName),
			cmds.DeleteProfile(profileName),
		)

	case cmds.CloseModalMsg:
		m.activeModal = nil
		if msg.Follow != nil {
			finalCmds = append(finalCmds, msg.Follow)
		}

	case cmds.CreateProfileMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetConfig)
		}

	case cmds.DeleteProfileMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetConfig)
		}
	}

	if m.activeModal != nil {
		var modalCmd tea.Cmd
		m.activeModal, modalCmd = m.activeModal.Update(msg)
		finalCmds = append(finalCmds, modalCmd)
	}

	// Update nested components
	var mainMenuCmd tea.Cmd
	m.components.MainMenu, mainMenuCmd = m.components.MainMenu.Update(msg)

	innerComponentsCmd := m.UpdateInnerComponent(m.activePage, msg)
	finalCmds = append(finalCmds, mainMenuCmd, innerComponentsCmd)

	return m, tea.Batch(finalCmds...)
}
```

(This drops the old inline profiles dedupe/sort block in `configSyncCmds` in favor of `m.allProfileNames()` from Step 1, and drops the now-unused `"stack-stitcher/src/utils"` import that only existed for that block — `utils` is re-imported in `AppModel.go` instead, where `allProfileNames` now lives.)

- [ ] **Step 3: Composite the modal on top of the screen in `View.go`**

Modify `src/model/View.go`. Change the import block from:

```go
import (
	"stack-stitcher/src/appstyles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)
```

(no change needed - already imports `lipgloss`). Replace the `View` function body from `layout := lipgloss.JoinVertical(...)` onward:

```go
		layout := lipgloss.JoinVertical(lipgloss.Left, sections...)
		rendered := appstyles.DocStyle.Render(layout)

		if m.activeModal != nil {
			rendered = m.renderWithModal(rendered)
		}

		v = tea.NewView(rendered)
		v.AltScreen = true
	}

	return v
}

// renderWithModal composites the active modal as a centered layer on top
// of the rest of the screen.
func (m AppModel) renderWithModal(base string) string {
	modalContent := m.activeModal.View().Content

	x := max(0, (m.config.terminalWidht-lipgloss.Width(modalContent))/2)
	y := max(0, (m.config.terminalHeight-lipgloss.Height(modalContent))/2)

	baseLayer := lipgloss.NewLayer(base)
	modalLayer := lipgloss.NewLayer(modalContent).X(x).Y(y).Z(1)

	return lipgloss.NewCompositor(baseLayer, modalLayer).Render()
}
```

The full file should now read:

```go
package model

import (
	"stack-stitcher/src/appstyles"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var errorBannerStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#FAFAFA")).
	Background(lipgloss.Color("#B33A3A")).
	Padding(0, 1)

func (m AppModel) View() tea.View {
	var v tea.View
	mainMenu := m.components.MainMenu.View().Content
	pageComponents, ok := m.pages[m.activePage]

	if ok {
		var contents []string

		for idx, _ := range pageComponents {
			contents = append(contents, pageComponents[idx].View().Content)
		}

		body := lipgloss.JoinHorizontal(lipgloss.Top, contents...)

		sections := []string{mainMenu, body}
		if m.lastError != "" {
			sections = []string{errorBannerStyle.Render("Error: " + m.lastError), mainMenu, body}
		}

		layout := lipgloss.JoinVertical(lipgloss.Left, sections...)
		rendered := appstyles.DocStyle.Render(layout)

		if m.activeModal != nil {
			rendered = m.renderWithModal(rendered)
		}

		v = tea.NewView(rendered)
		v.AltScreen = true
	}

	return v
}

// renderWithModal composites the active modal as a centered layer on top
// of the rest of the screen.
func (m AppModel) renderWithModal(base string) string {
	modalContent := m.activeModal.View().Content

	x := max(0, (m.config.terminalWidht-lipgloss.Width(modalContent))/2)
	y := max(0, (m.config.terminalHeight-lipgloss.Height(modalContent))/2)

	baseLayer := lipgloss.NewLayer(base)
	modalLayer := lipgloss.NewLayer(modalContent).X(x).Y(y).Z(1)

	return lipgloss.NewCompositor(baseLayer, modalLayer).Render()
}
```

- [ ] **Step 4: Verify everything builds and existing tests still pass**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: no errors; `ok stack-stitcher/src/utils ...` for the Task 1 tests; no test files elsewhere so those packages report `?   ...   [no test files]`

- [ ] **Step 5: Commit**

```bash
git add src/model/AppModel.go src/model/Update.go src/model/View.go
git commit -m "Wire a compositor-based modal overlay into AppModel"
```

---

### Task 7: Wire `n`/`d` keybindings into the Groups panel

**Files:**
- Modify: `src/components/ProfilesList.go`

**Interfaces:**
- Consumes: `cmds.OpenCreateProfileModal() tea.Cmd`, `cmds.OpenDeleteProfileModal(profileName string) tea.Cmd` (Task 2).
- Produces: nothing new — this is the last piece connecting the keybindings to the modal flow built in Tasks 2-6.

- [ ] **Step 1: Add the `n` and `d` cases**

Modify `src/components/ProfilesList.go`. Change the `tea.KeyPressMsg` case in `ProfileListModel.Update` from:

```go
	case tea.KeyPressMsg:
		switch msg.String() {
		case "space":
			if m.isFocused {
				m.listDelegate.activeIndex = m.list.GlobalIndex()
				m.list.SetDelegate(m.listDelegate)

				selectedItem := m.list.SelectedItem()
				selectedProfile, ok := selectedItem.(apptypes.ProfileListItem)

				if ok {
					selectedServiceCmd := cmds.SetSelectedProfile(string(selectedProfile))
					finalCmds = append(finalCmds, selectedServiceCmd)
				}
			}
		}
```

to:

```go
	case tea.KeyPressMsg:
		switch msg.String() {
		case "space":
			if m.isFocused {
				m.listDelegate.activeIndex = m.list.GlobalIndex()
				m.list.SetDelegate(m.listDelegate)

				selectedItem := m.list.SelectedItem()
				selectedProfile, ok := selectedItem.(apptypes.ProfileListItem)

				if ok {
					selectedServiceCmd := cmds.SetSelectedProfile(string(selectedProfile))
					finalCmds = append(finalCmds, selectedServiceCmd)
				}
			}

		case "n":
			if m.isFocused {
				finalCmds = append(finalCmds, cmds.OpenCreateProfileModal())
			}

		case "d":
			if m.isFocused {
				if selectedProfile, ok := m.list.SelectedItem().(apptypes.ProfileListItem); ok {
					finalCmds = append(finalCmds, cmds.OpenDeleteProfileModal(string(selectedProfile)))
				}
			}
		}
```

- [ ] **Step 2: Verify the package builds**

Run: `go build ./...`
Expected: no output, exit code 0

- [ ] **Step 3: Commit**

```bash
git add src/components/ProfilesList.go
git commit -m "Wire n/d keybindings to open create/delete profile modals"
```

---

### Task 8: Manual verification and README update

There's no automated way to drive Bubble Tea keypresses in this repo (no test harness for TUI interaction), so this task is a manual pass through both flows against a disposable copy of the existing demo fixture, followed by a README update to document the new keys.

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Build**

Run: `make build`
Expected: produces `dist/stack-stitcher` with no errors

- [ ] **Step 2: Set up a scratch compose file**

```bash
mkdir -p /tmp/stack-stitcher-manual-check
cp demo/fixtures/compose.yaml /tmp/stack-stitcher-manual-check/compose.yaml
cd /tmp/stack-stitcher-manual-check
/home/filipe/Documents/projects/tui/dist/stack-stitcher
```

- [ ] **Step 3: Exercise the create flow**

With the app running: Tab to focus the Groups panel on the Home page, press `n`, type `extra`, press `Enter`, use arrow keys + `Space` to check `cache`, press `Enter`.

Check:
- The modal closes and the Groups panel now shows an `extra` profile.
- `cat /tmp/stack-stitcher-manual-check/compose.yaml` shows `cache` tagged with `extra` and the other services' `profiles: ["core"] # core services` line/comment untouched.

- [ ] **Step 4: Exercise validation and cancel paths**

- Press `n`, `Enter` with an empty name → inline "can't be empty" message, modal stays open.
- Press `n`, type `core`, `Enter` → inline "already exists" message, modal stays open.
- Press `n`, type a valid name, `Enter`, then `Esc` on the checklist step → modal closes, no new profile appears, file unchanged.
- Press `n`, `Esc` on the name step → modal closes immediately.

- [ ] **Step 5: Exercise the delete flow**

Highlight the `extra` profile in the Groups panel, press `d` → confirmation prompt appears. Press `n` → prompt dismisses, profile still present. Press `d` again, then `y` → profile disappears from the Groups panel and `cache`'s `profiles:` key is gone from `compose.yaml` (its only tag was `extra`).

- [ ] **Step 6: Clean up the scratch directory**

```bash
rm -rf /tmp/stack-stitcher-manual-check
```

- [ ] **Step 7: Update the README**

In `README.md`, change:

```
Stack Stitcher is under **active development**. Compose parsing, navigation, and starting/stopping services (individually or as a whole profile) all work. Editing services, creating/deleting profiles from the TUI, and bootstrapping a compose file from scratch are still on the roadmap. Feedback, issues, and ideas are genuinely welcome and help shape where it goes next.
```

to:

```
Stack Stitcher is under **active development**. Compose parsing, navigation, starting/stopping services (individually or as a whole profile), and creating/deleting profiles all work. Editing services and bootstrapping a compose file from scratch are still on the roadmap. Feedback, issues, and ideas are genuinely welcome and help shape where it goes next.
```

And in the key bindings table, change:

```
| `p` | Pull | A profile or service panel focused |
| `x` | Remove | A profile or service panel focused |
| `q` / `Ctrl+C` | Quit | Everywhere |
```

to:

```
| `p` | Pull | A profile or service panel focused |
| `x` | Remove | A profile or service panel focused |
| `n` | Create a new profile | Groups panel focused |
| `d` | Delete the highlighted profile | Groups panel focused |
| `q` / `Ctrl+C` | Quit | Everywhere |
```

- [ ] **Step 8: Commit**

```bash
git add README.md
git commit -m "Document create/delete profile keybindings in the README"
```
