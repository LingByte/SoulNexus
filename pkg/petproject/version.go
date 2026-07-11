package petproject

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var semverRe = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-[\w.]+)?$`)

// BumpSemverPatch increments patch segment (1.0.0 → 1.0.1).
func BumpSemverPatch(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "1.0.0"
	}
	m := semverRe.FindStringSubmatch(v)
	if m == nil {
		return "1.0.1"
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return fmt.Sprintf("%d.%d.%d", major, minor, patch+1)
}

// UpdateSoulpetYAMLVersion rewrites or inserts version: line in soulpet.yaml text.
func UpdateSoulpetYAMLVersion(raw, version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return raw
	}
	lines := strings.Split(raw, "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "version:") {
			lines[i] = "version: \"" + version + "\""
			found = true
			break
		}
	}
	if !found {
		lines = append(lines, "version: \""+version+"\"")
	}
	return strings.Join(lines, "\n")
}

const ChangelogFile = "CHANGELOG.md"

// ApplyVersionBump updates soulpet.yaml semver and optional CHANGELOG.md in files map.
// Returns updated files and new semver string (empty if unchanged).
func ApplyVersionBump(files map[string]string, changeNote string, bump bool) (map[string]string, string) {
	if !bump && strings.TrimSpace(changeNote) == "" {
		return files, ""
	}
	next := make(map[string]string, len(files)+1)
	for k, v := range files {
		next[k] = v
	}
	current := "1.0.0"
	if raw, ok := next[SoulpetYamlFile]; ok {
		if meta, err := ParseSoulpetYAML(raw); err == nil && strings.TrimSpace(meta.Version) != "" {
			current = strings.TrimSpace(meta.Version)
		}
	}
	newVer := current
	if bump || strings.TrimSpace(changeNote) != "" {
		newVer = BumpSemverPatch(current)
	}
	if raw, ok := next[SoulpetYamlFile]; ok {
		next[SoulpetYamlFile] = UpdateSoulpetYAMLVersion(raw, newVer)
	}
	if note := strings.TrimSpace(changeNote); note != "" {
		next[ChangelogFile] = AppendChangelog(next[ChangelogFile], newVer, note)
	}
	return next, newVer
}

// AppendChangelog prepends an entry to CHANGELOG.md content.
func AppendChangelog(existing, version, note string) string {
	note = strings.TrimSpace(note)
	if note == "" {
		return existing
	}
	entry := fmt.Sprintf("## %s\n\n- %s\n\n", version, note)
	if strings.TrimSpace(existing) == "" {
		return "# Changelog\n\n" + entry
	}
	if strings.HasPrefix(existing, "# Changelog") {
		parts := strings.SplitN(existing, "\n", 2)
		if len(parts) == 2 {
			return parts[0] + "\n\n" + entry + strings.TrimLeft(parts[1], "\n")
		}
	}
	return "# Changelog\n\n" + entry + existing
}
