package version
import "testing"
func TestResolvedDefaults(t *testing.T) {
  if GetGoVersion() == "unknown" { t.Fatalf("go=%s", GetGoVersion()) }
  if GetGitCommit() == "unknown" { t.Fatalf("commit=%s", GetGitCommit()) }
  if GetBuildTime() == "unknown" { t.Fatalf("built=%s", GetBuildTime()) }
  t.Log(GetVersionInfo())
}
