package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestDescribeBuildOrigins(t *testing.T) {
	checkout := &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}, Settings: []debug.BuildSetting{
		{Key: "vcs.revision", Value: "0123456789abcdef"}, {Key: "vcs.modified", Value: "true"},
	}}
	pseudoVersion := *checkout
	pseudoVersion.Main.Version = "v0.4.1-0.20261003114800-3b47c7c18436+dirty"
	for _, test := range []struct {
		name, version, commit, want string
		info                        *debug.BuildInfo
	}{
		{name: "release", version: "0.5.0", commit: "abcdef0123456789", want: "v0.5.0 (commit abcdef012345)"},
		{name: "snapshot", version: "0.5.0-SNAPSHOT-abcdef", want: "v0.5.0-SNAPSHOT-abcdef"},
		{name: "module install", info: &debug.BuildInfo{Main: debug.Module{Version: "v0.5.0"}}, want: "v0.5.0"},
		{name: "checkout", info: checkout, want: "devel (commit 0123456789ab-dirty)"},
		{name: "checkout inferred version", info: &pseudoVersion, want: "devel (commit 0123456789ab-dirty)"},
		{name: "no build metadata", want: "devel"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := describe(test.info, test.version, test.commit); got != test.want {
				t.Fatalf("version = %q, want %q", got, test.want)
			}
		})
	}
}
