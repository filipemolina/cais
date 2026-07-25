# Bootstrap a Compose File Implementation Plan (Completed)

> **Historical execution record — implemented.** All numbered tasks in this
> plan are complete. The unchecked boxes below preserve the original plan and
> are **not** a current backlog or workflow instruction. See
> [README](../../../README.md), [Design](../../DESIGN.md), and
> [TODO](../../../TODO.md) for current guidance.

**Goal:** Let the user create a fresh `compose.yaml` in the current working directory from inside the TUI, with an optional pre-filled first service, so a brand-new directory can be driven end-to-end (read → group → operate → log) without ever leaving the app.

**Architecture:** Reuse the `activeModal` overlay, key-routing bypass, and `CloseModalMsg` follow-up mechanism that create/delete-profiles already added. One new leaf modal (`CreateComposeFileModal`) does a two-step form (filename → optional first service), then emits a follow-up `cmds.CreateComposeFile` that calls a new pure function in `src/utils/ProfileTags.go` (`WriteNewComposeFile`). The write uses the same `yaml.v3` Node API and reuses the existing `writeComposeNode` helper so the produced file is already in the format `AddProfileTag`/`RemoveProfileTag` operate on. A new sentinel `utils.ErrNoComposeFile` distinguishes "file is missing — offer to bootstrap" from other `GetConfig` errors, and `AppModel.Update`'s `GetConfigMsg` handler auto-opens the modal whenever that sentinel comes through — no keypress required.

**Tech Stack:** Go 1.26, Bubble Tea v2 / Bubbles v2 / Lip Gloss v2 (`charm.land/...`), `github.com/compose-spec/compose-go/v2/types`, `gopkg.in/yaml.v3` (already a direct dep, added in the previous plan).

## Global Constraints

- Module: `stack-stitcher`, Go 1.26.4 (`go.mod:1-3`).
- Follow the existing per-component `Init`/`Update`/`View` pattern; one exported constructor per component file (see `src/components/CreateComposeFileModal.go` mirroring `src/components/ProfileNameModal.go`).
- All cross-component communication goes through `src/cmds/` message types dispatched as `tea.Cmd` — never call between components directly.
- Errors from the file-system action surface via `AppModel.lastError` and the existing red banner in `src/model/View.go` (`errorBannerStyle`) — do not invent a second error-display mechanism.
- Form-validation errors stay inline inside the modal — they are not IO/docker errors and must not reach `m.lastError`.
- No new test framework: extend the existing `src/utils/ProfileTags_test.go` (added by the previous plan); components, cmds, and wiring follow the existing untested convention and are checked with `go build ./...` plus the manual verification in Task 5.
- Design reference: `docs/superpowers/specs/2026-07-23-bootstrap-compose-file-design.md`.

---

### Task 1: Sentinel error + `WriteNewComposeFile` in `src/utils`

**Files:**
- Modify: `src/utils/GetComposeFileName.go`
- Modify: `src/utils/ProfileTags.go`
- Modify: `src/utils/ProfileTags_test.go`

**Interfaces:**
- Produces (consumed by Task 2's `cmds.CreateComposeFile` and Task 4's `AppModel.Update`):
  - `var ErrNoComposeFile = errors.New("no compose file found in the current directory")` — exported from `src/utils/GetComposeFileName.go`, returned by `GetComposeFileName()` when no candidate file is present.
  - `func WriteNewComposeFile(fileName string, serviceName string, image string) error` — atomic, refuses to overwrite an existing file; writes a new `compose.yaml` with `services:` mapping and optional one-service seed.

- [ ] **Step 1: Add the sentinel error to `GetComposeFileName.go`**

Replace the body of `src/utils/GetComposeFileName.go` with:

```go
package utils

import (
	"errors"
	"fmt"
	"os"
)

// ErrNoComposeFile is returned by GetComposeFileName when none of the
// candidate file names exist in the current directory. The bootstrap flow
// uses it to distinguish "no file yet" from other load errors and offer to
// create one.
var ErrNoComposeFile = errors.New("no compose file found in the current directory")

func GetComposeFileName() (string, error) {
	files, err := os.ReadDir(".")

	configFileNames := []string{
		"compose.yaml",
		"compose.yml",
		"docker-compose.yaml",
		"docker-compose.yml",
	}

	if err != nil {
		return "", fmt.Errorf("failed reading the current directory: %w", err)
	}

	curDirFileNames := make(map[string]struct{})

	for _, file := range files {
		if !file.IsDir() {
			curDirFileNames[file.Name()] = struct{}{}
		}
	}

	for _, fileName := range configFileNames {
		if _, ok := curDirFileNames[fileName]; ok {
			return fileName, nil
		}
	}

	return "", ErrNoComposeFile
}
```

- [ ] **Step 2: Add `WriteNewComposeFile` to `ProfileTags.go`**

Append the following function to `src/utils/ProfileTags.go` (it sits naturally with the other disk-write helpers; the existing `writeComposeNode` is reused for re-encoding):

```go
// WriteNewComposeFile creates a brand-new compose file at fileName with a
// top-level services mapping, optionally pre-seeded with one service. It
// refuses to overwrite an existing file: the caller is expected to have
// already shown a validation error in the modal in that case, so we surface
// os.ErrExist to make the failure mode explicit.
func WriteNewComposeFile(fileName string, serviceName string, image string) error {
	if _, err := os.Stat(fileName); err == nil {
		return fmt.Errorf("refusing to overwrite existing %s: %w", fileName, os.ErrExist)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("checking %s: %w", fileName, err)
	}

	doc := &yaml.Node{Kind: yaml.MappingNode}

	servicesValue := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	if serviceName != "" {
		serviceMapping := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		serviceMapping.Content = append(serviceMapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: serviceName},
			&yaml.Node{
				Kind: yaml.MappingNode,
				Tag:   "!!map",
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "image"},
					{Kind: yaml.ScalarNode, Value: image, Tag: "!!str"},
				},
			},
		)
		servicesValue.Content = append(servicesValue.Content, serviceMapping.Content...)
	}

	doc.Content = []*yaml.Node{
		{Kind: yaml.ScalarNode, Value: "services"},
		servicesValue,
	}

	return writeComposeNode(fileName, doc)
}
```

(`writeComposeNode` already exists in the same file from the create/delete-profiles work — Task 1 of that plan.)

- [ ] **Step 3: Add tests to `ProfileTags_test.go`**

Append the following cases to `src/utils/ProfileTags_test.go`. They exercise the three guarantees: empty file is valid, one-service seed round-trips, and we refuse to overwrite.

```go
func TestWriteNewComposeFile_Empty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := WriteNewComposeFile(path, "", ""); err != nil {
		t.Fatalf("WriteNewComposeFile: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	if !strings.Contains(string(raw), "services:") {
		t.Errorf("expected output to contain a top-level services: key, got:\n%s", raw)
	}
}

func TestWriteNewComposeFile_OneService(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := WriteNewComposeFile(path, "app", "nginx:alpine"); err != nil {
		t.Fatalf("WriteNewComposeFile: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading result file: %v", err)
	}

	var doc struct {
		Services map[string]struct {
			Image string `yaml:"image"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parsing result file: %v", err)
	}

	got, ok := doc.Services["app"]
	if !ok {
		t.Fatalf("expected an app service in the output, got:\n%s", raw)
	}
	if got.Image != "nginx:alpine" {
		t.Errorf("app image = %q, want %q", got.Image, "nginx:alpine")
	}
}

func TestWriteNewComposeFile_ThenAddProfileTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "compose.yaml")

	if err := WriteNewComposeFile(path, "app", "nginx:alpine"); err != nil {
		t.Fatalf("WriteNewComposeFile: %v", err)
	}

	if err := AddProfileTag(path, "core", []string{"app"}); err != nil {
		t.Fatalf("AddProfileTag on freshly written file: %v", err)
	}

	got := readServiceProfiles(t, path, "app")
	want := []string{"core"}
	if !slices.Equal(got, want) {
		t.Errorf("app profiles = %v, want %v", got, want)
	}
}

func TestWriteNewComposeFile_RefusesExisting(t *testing.T) {
	path := writeFixture(t, baseFixture)

	err := WriteNewComposeFile(path, "app", "nginx:alpine")
	if err == nil {
		t.Fatalf("expected error when target file already exists")
	}
	if !errors.Is(err, os.ErrExist) {
		t.Errorf("expected wrapped os.ErrExist, got: %v", err)
	}
}
```

Add the `errors` import alongside the existing imports in `ProfileTags_test.go`:

```go
import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)
```

- [ ] **Step 4: Verify the new tests fail and the old tests still pass**

Run: `go test ./src/utils/... -v`
Expected: the four new tests FAIL (function not defined / sentinel not present); the existing five create/delete-profiles tests still PASS.

- [ ] **Step 5: Verify everything now passes**

Run: `go test ./src/utils/... -v`
Expected: all nine tests PASS.

- [ ] **Step 6: Commit**

```bash
git add src/utils/GetComposeFileName.go src/utils/ProfileTags.go src/utils/ProfileTags_test.go
git commit -m "Add ErrNoComposeFile sentinel and WriteNewComposeFile helper"
```

---

### Task 2: `cmds` message for compose-file creation

**Files:**
- Create: `src/cmds/CreateComposeFile.go`

**Interfaces:**
- Produces (consumed by Task 4's `AppModel.Update` and Task 3's modal):
  - `type CreateComposeFileMsg struct { Err error }` / `func CreateComposeFile(fileName string, serviceName string, image string) tea.Cmd`

  The modal itself is constructed directly inside `AppModel.Update`'s `GetConfigMsg` handler (no `OpenCreateComposeFileModal` cmd is needed — the trigger is the sentinel error, not a user keypress).

- [ ] **Step 1: Create the action cmd**

`src/cmds/CreateComposeFile.go`:

```go
package cmds

import (
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
)

type CreateComposeFileMsg struct {
	Err error
}

// CreateComposeFile writes a brand-new compose file at fileName in the
// current working directory, optionally pre-seeded with one service. Empty
// serviceName/image means "create an empty services: mapping".
func CreateComposeFile(fileName string, serviceName string, image string) tea.Cmd {
	return func() tea.Msg {
		return CreateComposeFileMsg{Err: utils.WriteNewComposeFile(fileName, serviceName, image)}
	}
}
```

- [ ] **Step 2: Verify the package builds**

Run: `go build ./...`
Expected: no output, exit code 0.

- [ ] **Step 3: Commit**

```bash
git add src/cmds/CreateComposeFile.go
git commit -m "Add cmds for compose-file creation"
```

---

### Task 3: `CreateComposeFileModal` component (two-step form)

**Files:**
- Create: `src/components/CreateComposeFileModal.go`

**Interfaces:**
- Consumes: `cmds.CloseModal` (already in place from the previous plan), `cmds.CreateComposeFile` (Task 2).
- Produces: `func CreateComposeFileModal() tea.Model` — consumed by Task 4's `AppModel.Update` (`GetConfigMsg` handler with `ErrNoComposeFile`).

- [ ] **Step 1: Write the component**

`src/components/CreateComposeFileModal.go`:

```go
package components

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"stack-stitcher/src/appstyles"
	"stack-stitcher/src/cmds"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type createStep int

const (
	stepFilename createStep = iota
	stepAddServicePrompt
	stepServiceFields
)

type CreateComposeFileModalModel struct {
	step        createStep
	filename    textinput.Model
	serviceName textinput.Model
	image       textinput.Model
	errMsg      string
}

func (m CreateComposeFileModalModel) Init() tea.Cmd {
	return nil
}

func (m CreateComposeFileModalModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch m.step {
		case stepFilename:
			return m.updateFilename(msg)
		case stepAddServicePrompt:
			return m.updateAddServicePrompt(msg)
		case stepServiceFields:
			return m.updateServiceFields(msg)
		}
	}

	// Forward non-key messages (e.g. WindowSizeMsg) to the active input so
	// the cursor keeps blinking and resize still flows.
	switch m.step {
	case stepFilename:
		var cmd tea.Cmd
		m.filename, cmd = m.filename.Update(msg)
		return m, cmd
	case stepServiceFields:
		if m.serviceName.Focused() {
			var cmd tea.Cmd
			m.serviceName, cmd = m.serviceName.Update(msg)
			return m, cmd
		}
		var cmd tea.Cmd
		m.image, cmd = m.image.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m CreateComposeFileModalModel) updateFilename(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, cmds.CloseModal(nil)
	case "enter":
		name := strings.TrimSpace(m.filename.Value())
		if name == "" {
			m.errMsg = "Filename can't be empty"
			return m, nil
		}
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			m.errMsg = "Filename must end in .yaml or .yml"
			return m, nil
		}
		if _, err := os.Stat(name); err == nil {
			m.errMsg = fmt.Sprintf("%s already exists in this directory", name)
			return m, nil
		} else if !os.IsNotExist(err) {
			m.errMsg = fmt.Sprintf("Can't stat %s: %v", name, err)
			return m, nil
		}
		m.errMsg = ""
		m.step = stepAddServicePrompt
		return m, nil
	}

	var cmd tea.Cmd
	m.filename, cmd = m.filename.Update(msg)
	return m, cmd
}

func (m CreateComposeFileModalModel) updateAddServicePrompt(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, cmds.CloseModal(nil)
	case "y", "Y":
		m.errMsg = ""
		m.step = stepServiceFields
		return m, m.serviceName.Focus()
	case "n", "N":
		return m, cmds.CloseModal(cmds.CreateComposeFile(strings.TrimSpace(m.filename.Value()), "", ""))
	}

	return m, nil
}

func (m CreateComposeFileModalModel) updateServiceFields(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		return m, cmds.CloseModal(nil)
	case "tab":
		if m.serviceName.Focused() {
			m.serviceName.Blur()
			return m, m.image.Focus()
		}
		m.image.Blur()
		return m, m.serviceName.Focus()
	case "enter":
		name := strings.TrimSpace(m.serviceName.Value())
		image := strings.TrimSpace(m.image.Value())
		if name == "" {
			m.errMsg = "Service name can't be empty"
			return m, nil
		}
		if image == "" {
			m.errMsg = "Image can't be empty (e.g. nginx:alpine)"
			return m, nil
		}
		if !isValidServiceName(name) {
			m.errMsg = fmt.Sprintf("%q is not a valid service name", name)
			return m, nil
		}
		return m, cmds.CloseModal(cmds.CreateComposeFile(strings.TrimSpace(m.filename.Value()), name, image))
	}

	if m.serviceName.Focused() {
		var cmd tea.Cmd
		m.serviceName, cmd = m.serviceName.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.image, cmd = m.image.Update(msg)
	return m, cmd
}

func isValidServiceName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return true
}

func (m CreateComposeFileModalModel) View() tea.View {
	wrapper := lipgloss.NewStyle().
		Padding(1, 2).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(appstyles.PrimaryColor).
		Background(appstyles.PanelBackgroundColor)

	errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#B33A3A"))
	var lines []string

	switch m.step {
	case stepFilename:
		lines = []string{
			"New compose file",
			"Filename (in the current directory):",
			m.filename.View(),
		}
	case stepAddServicePrompt:
		lines = []string{
			fmt.Sprintf("Creating %s", filepath.Base(strings.TrimSpace(m.filename.Value()))),
			"Add a first service? (y/n)",
		}
	case stepServiceFields:
		lines = []string{
			fmt.Sprintf("Creating %s", filepath.Base(strings.TrimSpace(m.filename.Value()))),
			"Service name:",
			m.serviceName.View(),
			"Image:",
			m.image.View(),
		}
	}

	if m.errMsg != "" {
		lines = append(lines, errStyle.Render(m.errMsg))
	}

	return tea.NewView(wrapper.Render(lipgloss.JoinVertical(lipgloss.Left, lines...)))
}

// CreateComposeFileModal walks the user through creating a brand-new compose
// file in the current directory: a filename (with a sane default and basic
// validation) and an optional one-service seed. Esc cancels the whole flow
// at any point - the file is never half-created.
func CreateComposeFileModal() tea.Model {
	filename := textinput.New()
	filename.Placeholder = "compose.yaml"
	filename.SetWidth(40)
	filename.SetValue("compose.yaml")
	filename.CursorEnd()
	filename.Focus()

	serviceName := textinput.New()
	serviceName.Placeholder = "e.g. web"
	serviceName.SetWidth(30)
	serviceName.Focus()

	image := textinput.New()
	image.Placeholder = "e.g. nginx:alpine"
	image.SetWidth(30)

	return CreateComposeFileModalModel{
		step:        stepFilename,
		filename:    filename,
		serviceName: serviceName,
		image:       image,
	}
}
```

- [ ] **Step 2: Verify the package builds**

Run: `go build ./...`
Expected: no output, exit code 0.

- [ ] **Step 3: Commit**

```bash
git add src/components/CreateComposeFileModal.go
git commit -m "Add CreateComposeFileModal two-step form"
```

---

### Task 4: Wire the modal into `AppModel` via the `ErrNoComposeFile` sentinel

**Files:**
- Modify: `src/model/Update.go`

**Interfaces:**
- Consumes: `components.CreateComposeFileModal` (Task 3); `cmds.CreateComposeFileMsg`, `utils.ErrNoComposeFile` (Tasks 1-2).
- Produces: nothing new — the `activeModal` field and key-routing bypass already exist from the create/delete-profiles work.

- [ ] **Step 1: Update imports and add the new message cases in `Update.go`**

Edit `src/model/Update.go`. Change the import block from:

```go
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
```

to:

```go
import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"slices"
	"stack-stitcher/src/cmds"
	"stack-stitcher/src/components"
	"stack-stitcher/src/utils"

	tea "charm.land/bubbletea/v2"
	"github.com/compose-spec/compose-go/v2/types"
)
```

Modify the existing `GetConfigMsg` handler. The current code is:

```go
	case cmds.GetConfigMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
			break
		}

		m.config.configFileName = msg.FileName
		m.config.configProject = msg.Project
		finalCmds = append(finalCmds, m.configSyncCmds()...)
```

Replace it with:

```go
	case cmds.GetConfigMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
			// No compose file in the current directory: offer to create
			// one in place. The error banner is still set above, so an
			// Esc from the modal leaves a visible explanation.
			if errors.Is(msg.Err, utils.ErrNoComposeFile) {
				m.activeModal = components.CreateComposeFileModal()
			}
			break
		}

		m.config.configFileName = msg.FileName
		m.config.configProject = msg.Project
		finalCmds = append(finalCmds, m.configSyncCmds()...)
```

Add the `CreateComposeFileMsg` case inside the existing type switch, right after `DeleteProfileMsg`:

```go
	case cmds.CreateComposeFileMsg:
		if msg.Err != nil {
			m.lastError = msg.Err.Error()
		} else {
			m.lastError = ""
			finalCmds = append(finalCmds, cmds.GetConfig)
		}
```

- [ ] **Step 2: Verify everything builds and existing tests still pass**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: no errors; `ok stack-stitcher/src/utils ...` for the Task 1 tests (now nine cases); no test files elsewhere so those packages report `?   ...   [no test files]`.

- [ ] **Step 3: Commit**

```bash
git add src/model/Update.go
git commit -m "Auto-open the bootstrap modal when ErrNoComposeFile surfaces from GetConfig"
```

---

### Task 5: Manual verification and README update

There's no automated way to drive Bubble Tea keypresses in this repo (no test harness for TUI interaction), so this task is a manual pass through the bootstrap flow against a disposable empty directory, followed by a README update to document the new key.

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Build**

Run: `make build`
Expected: produces `dist/stack-stitcher` with no errors.

- [ ] **Step 2: Set up a scratch empty directory**

```bash
rm -rf /tmp/stack-stitcher-bootstrap-check
mkdir -p /tmp/stack-stitcher-bootstrap-check
cd /tmp/stack-stitcher-bootstrap-check
/home/filipe/Documents/projects/tui/dist/stack-stitcher
```

Expected: the app launches and the bootstrap modal appears **automatically** — no keypress required. A centered modal titled "New compose file" with a text input pre-filled with `compose.yaml` and the cursor at the end. (The red error banner with `Error: no compose file found in the current directory` is also set under the hood and would become visible if the user Escs.)

- [ ] **Step 3: Esc the modal — confirm the banner underneath**

Press `Esc`. Expected: modal closes, red error banner reads `Error: no compose file found in the current directory`. The TUI is otherwise empty (no Groups, no Services).

- [ ] **Step 4: Re-run to bring the modal back**

Quit (`q`/`Ctrl+C`) and re-run the binary. Expected: modal reappears on startup.

- [ ] **Step 5: Exercise the filename step**

- Press `Enter` immediately (accepting the default) → modal advances to "Add a first service? (y/n)".
- Press `Esc` → modal closes, no file is created. `ls` confirms the directory is still empty.
- Re-run, type `compose.json` (or any name not ending in `.yaml`/`.yml`), press `Enter` → inline "Filename must end in .yaml or .yml" message, modal stays on step 1.
- Clear, type `docker-compose.yaml` (which doesn't exist here), press `Enter` → advances to the service prompt. Press `Esc`.
- Re-run, clear the default with backspace, press `Enter` → inline "Filename can't be empty". Type `compose.yaml`, press `Enter` → advances to the service prompt.

- [ ] **Step 6: Exercise the "add a first service?" prompt**

- Press `y` → modal advances to two text inputs: "Service name:" (focused) and "Image:".
- Press `Tab` → focus moves to "Image:".
- Press `Tab` again → focus moves back to "Service name:".
- Type `web` for the name, `Tab`, type `nginx:alpine`, press `Enter`.

Expected: modal closes; `ls` shows `compose.yaml`; `cat compose.yaml` shows a top-level `services:` mapping with `web:` and `image: nginx:alpine`. The TUI re-renders with `web` listed in the Groups / Services panels after the auto-refresh.

- [ ] **Step 7: Exercise the empty-file path**

- Delete the file: `rm /tmp/stack-stitcher-bootstrap-check/compose.yaml`.
- Run the binary again. Accept the default filename, press `Enter`, press `n` to skip the first service.

Expected: modal closes; `cat compose.yaml` shows just `services:` with no children (or an empty mapping — both load fine through `compose-go`).

- [ ] **Step 8: Exercise the validation error path on the service fields**

- Delete and re-run. Accept the default, `Enter`, `y` → service fields.
- Press `Enter` immediately (both empty) → inline "Service name can't be empty".
- Type `web app` (space) → press `Enter` → inline "<name> is not a valid service name" (whitespace disallowed).
- Type `web`, `Tab`, leave image empty, press `Enter` → inline "Image can't be empty (e.g. nginx:alpine)".
- Press `Esc` → modal closes; no file is written.

- [ ] **Step 9: Exercise the "file already exists" guard**

The dedicated overwrite-guard test is already covered by the unit test in Task 1 (`TestWriteNewComposeFile_RefusesExisting`). The modal-side guard is the same `os.Stat` check, so the manual pass is satisfied as long as the modal never half-creates. To smoke-test it from the UI:

```bash
rm -rf /tmp/stack-stitcher-bootstrap-check
mkdir -p /tmp/stack-stitcher-bootstrap-check
cd /tmp/stack-stitcher-bootstrap-check
touch somethingelse.yaml
/home/filipe/Documents/projects/tui/dist/stack-stitcher
```

Expected: bootstrap modal appears (because `somethingelse.yaml` doesn't match any candidate name). Type `somethingelse.yaml` in the filename input, press `Enter` → inline `somethingelse.yaml already exists in this directory` error, modal stays on step 1. Type `compose.yaml`, press `Enter` → advances normally. Press `Esc`.

- [ ] **Step 10: Clean up the scratch directory**

```bash
rm -rf /tmp/stack-stitcher-bootstrap-check
```

- [ ] **Step 11: Update the README**

In `README.md`, change:

```
Stack Stitcher is under **active development**. Compose parsing, navigation, starting/stopping services (individually or as a whole profile), and creating/deleting profiles all work. Editing services and bootstrapping a compose file from scratch are still on the roadmap. Feedback, issues, and ideas are genuinely welcome and help shape where it goes next.
```

to:

```
Stack Stitcher is under **active development**. Compose parsing, navigation, starting/stopping services (individually or as a whole profile), creating/deleting profiles, and bootstrapping a new compose file from inside the TUI all work. Editing existing services is still on the roadmap. Feedback, issues, and ideas are genuinely welcome and help shape where it goes next.
```

In the key bindings table, change:

```
| `n` | Create a new profile | Groups panel focused |
| `d` | Delete the highlighted profile | Groups panel focused |
| `l` | View live logs (streaming overlay) | A profile or service panel focused |
```

to:

```
| `n` | Create a new profile | Groups panel focused |
| `d` | Delete the highlighted profile | Groups panel focused |
| `l` | View live logs (streaming overlay) | A profile or service panel focused |
```

(No new keybinding row — the bootstrap modal is opened automatically.)

- [ ] **Step 12: Commit**

```bash
git add README.md
git commit -m "Document the auto-prompt bootstrap compose-file flow in the README"
```
