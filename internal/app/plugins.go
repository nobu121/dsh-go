package app

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	webProfileName     = "web"
	officialPluginPref = "@deepseek-ai/"
)

type userPlugin struct {
	Name    string
	Version string
}

func webProfileDir(home string) string {
	return filepath.Join(home, "profiles", webProfileName)
}

func webProfileManifestPath(home string) string {
	return filepath.Join(webProfileDir(home), "package.json")
}

func webProfilePatchPath(home string) string {
	return filepath.Join(webProfileDir(home), "cordis.patch.yml")
}

func isOfficialPlugin(name string) bool {
	return strings.HasPrefix(name, officialPluginPref)
}

func pluginMentioned(text, name string) bool {
	return name != "" && text != "" && strings.Contains(text, name)
}

func listUserPlugins(home string) []userPlugin {
	doc, _, err := readProfileManifest(home)
	if err != nil {
		return nil
	}
	deps := profileDependencies(doc)
	seen := map[string]bool{}
	var out []userPlugin
	add := func(name string) {
		if name == "" || isOfficialPlugin(name) || seen[name] {
			return
		}
		seen[name] = true
		out = append(out, userPlugin{Name: name, Version: deps[name]})
	}
	for _, name := range profileBundles(doc) {
		add(name)
	}
	var extra []string
	for name := range deps {
		if !seen[name] && !isOfficialPlugin(name) {
			extra = append(extra, name)
		}
	}
	sort.Strings(extra)
	for _, name := range extra {
		add(name)
	}
	return out
}

func disableProfileBundles(home string, names []string) error {
	var disable []string
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || isOfficialPlugin(name) {
			continue
		}
		disable = append(disable, name)
	}
	if len(disable) == 0 {
		return nil
	}
	patchPath := webProfilePatchPath(home)
	for _, name := range disable {
		ids := pluginPatchRowIDs(home, name)
		if len(ids) == 0 {
			return fmt.Errorf("no loader rows for %s", name)
		}
		for _, id := range ids {
			if err := appendPatchDisable(patchPath, id); err != nil {
				return err
			}
		}
	}
	return nil
}

func recoverPrepProgress(home, blamedText string) PrepProgress {
	ui := currentUI()
	plugins := listUserPlugins(home)
	var selected, rest []PrepOfferItem
	for _, p := range plugins {
		item := PrepOfferItem{
			Kind:     "plugin",
			Name:     p.Name,
			Version:  p.Version,
			Selected: pluginMentioned(blamedText, p.Name),
		}
		if item.Selected {
			selected = append(selected, item)
		} else {
			rest = append(rest, item)
		}
	}
	return PrepProgress{
		Stage:   prepRecoverStage,
		Message: ui.RecoverFail,
		Action:  ui.DisableRestart,
		Detail:  strings.TrimSpace(blamedText),
		Items:   append(selected, rest...),
	}
}

func parseRecoverNames(data any) []string {
	switch v := data.(type) {
	case string:
		var names []string
		for _, line := range strings.Split(v, "\n") {
			if name := strings.TrimSpace(line); name != "" {
				names = append(names, name)
			}
		}
		return names
	case []any:
		var names []string
		for _, item := range v {
			s, _ := item.(string)
			if name := strings.TrimSpace(s); name != "" {
				names = append(names, name)
			}
		}
		return names
	case []string:
		var names []string
		for _, s := range v {
			if name := strings.TrimSpace(s); name != "" {
				names = append(names, name)
			}
		}
		return names
	default:
		return nil
	}
}

func readProfileManifest(home string) (map[string]any, []byte, error) {
	raw, err := os.ReadFile(webProfileManifestPath(home))
	if err != nil {
		return nil, nil, err
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, raw, err
	}
	if doc == nil {
		return nil, raw, fmt.Errorf("profile manifest is empty")
	}
	return doc, raw, nil
}

func profileDependencies(doc map[string]any) map[string]string {
	raw, _ := doc["dependencies"].(map[string]any)
	if raw == nil {
		return nil
	}
	out := make(map[string]string, len(raw))
	for name, ver := range raw {
		out[name] = fmt.Sprint(ver)
	}
	return out
}

func profileBundles(doc map[string]any) []string {
	dsh, _ := doc["dsh"].(map[string]any)
	if dsh == nil {
		return nil
	}
	profile, _ := dsh["profile"].(map[string]any)
	if profile == nil {
		return nil
	}
	raw, _ := profile["bundles"].([]any)
	var out []string
	for _, item := range raw {
		name, _ := item.(string)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

func pluginPatchRowIDs(home, name string) []string {
	root := filepath.Join(webProfileDir(home), "node_modules", filepath.FromSlash(name))
	paths := []string{filepath.Join(root, "cordis.patch.yml")}
	if b, err := os.ReadFile(filepath.Join(root, "package.json")); err == nil {
		var doc struct {
			DSH struct {
				Bundle struct {
					Patch string `json:"patch"`
				} `json:"bundle"`
			} `json:"dsh"`
		}
		if json.Unmarshal(b, &doc) == nil && doc.DSH.Bundle.Patch != "" {
			paths = append(paths, filepath.Join(root, filepath.FromSlash(doc.DSH.Bundle.Patch)))
		}
	}
	seen := map[string]bool{}
	var ids []string
	for _, path := range paths {
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, id := range scanPatchInsertIDs(string(raw)) {
			if seen[id] {
				continue
			}
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

func scanPatchInsertIDs(text string) []string {
	var ids []string
	inInsert := false
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- insert:") {
			inInsert = true
			continue
		}
		if strings.HasPrefix(trimmed, "- ") && !strings.HasPrefix(strings.TrimSpace(strings.TrimPrefix(trimmed, "-")), "id:") {
			if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
				inInsert = false
			}
		}
		if !inInsert {
			continue
		}
		id, ok := parseYAMLID(trimmed)
		if !ok {
			id, ok = parseYAMLID(strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
		}
		if ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func parseYAMLID(line string) (string, bool) {
	line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
	key, val, ok := strings.Cut(line, ":")
	if !ok || strings.TrimSpace(key) != "id" {
		return "", false
	}
	id := strings.Trim(strings.TrimSpace(val), `"'`)
	if id == "" || strings.ContainsAny(id, " \t") {
		return "", false
	}
	for _, c := range id {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '.' || c == '-') {
			return "", false
		}
	}
	return id, true
}

func appendPatchDisable(path, rowID string) error {
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	text := string(raw)
	if patchAlreadyDisables(text, rowID) {
		return nil
	}
	block := "- id: " + rowID + "\n  disabled: true\n"
	core := strings.TrimSpace(stripYAMLComments(text))
	if core == "" {
		return os.WriteFile(path, []byte(block), 0o644)
	}
	if core == "[]" || core == "[ ]" {
		commented := emptyListPlaceholder.ReplaceAllString(text, "# []\n")
		if !strings.HasSuffix(commented, "\n") {
			commented += "\n"
		}
		return os.WriteFile(path, []byte(commented+block), 0o644)
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	return os.WriteFile(path, []byte(text+block), 0o644)
}

func patchAlreadyDisables(text, rowID string) bool {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		id, ok := parseYAMLID(strings.TrimSpace(line))
		if !ok || id != rowID || i+1 >= len(lines) {
			continue
		}
		if strings.TrimSpace(lines[i+1]) == "disabled: true" {
			return true
		}
	}
	return false
}

func stripYAMLComments(text string) string {
	var b strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

var emptyListPlaceholder = regexp.MustCompile(`(?m)^[ \t]*\[[ \t]*\][ \t]*(?:#.*)?(?:\r?\n|$)`)
