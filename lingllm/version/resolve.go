package version

import (
	"os/exec"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

func init() {
	resolveVersionDefaults()
}

func resolveVersionDefaults() {
	if strings.TrimSpace(GoVersion) == "" || GoVersion == "unknown" {
		GoVersion = runtime.Version()
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if isUnknown(GitCommit) && strings.TrimSpace(s.Value) != "" {
					GitCommit = shortRevision(s.Value)
				}
			case "vcs.time":
				if isUnknown(BuildTime) && strings.TrimSpace(s.Value) != "" {
					BuildTime = s.Value
				}
			}
		}
	}
	if isUnknown(GitCommit) {
		if v := gitLine("rev-parse", "--short", "HEAD"); v != "" {
			GitCommit = v
		}
	}
	if isUnknown(BuildTime) {
		if v := gitLine("log", "-1", "--format=%ci"); v != "" {
			BuildTime = v
		} else {
			BuildTime = time.Now().UTC().Format(time.RFC3339)
		}
	}
}

func isUnknown(v string) bool {
	return strings.TrimSpace(v) == "" || v == "unknown"
}

func shortRevision(rev string) string {
	rev = strings.TrimSpace(rev)
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}

func gitLine(args ...string) string {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
