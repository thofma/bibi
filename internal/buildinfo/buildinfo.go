// Package buildinfo identifies release, module-installed and checkout builds.
package buildinfo

import (
	"runtime/debug"
	"strings"
)

// Version and Commit are supplied by the release linker flags.
var Version, Commit string

func String() string {
	info, _ := debug.ReadBuildInfo()
	return describe(info, Version, Commit)
}

func describe(info *debug.BuildInfo, version, commit string) string {
	dirty := false
	if info != nil {
		checkout := false
		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				checkout = true
				if commit == "" {
					commit = setting.Value
				}
			case "vcs.modified":
				dirty = setting.Value == "true"
			}
		}
		// Recent Go versions infer a module pseudo-version for local builds.
		// VCS settings identify those builds, unlike go install module@version.
		if version == "" && !checkout && info.Main.Version != "(devel)" {
			version = info.Main.Version
		}
	}
	if version == "" {
		version = "devel"
	} else if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if len(commit) > 12 {
		commit = commit[:12]
	}
	if dirty {
		commit += "-dirty"
	}
	if commit != "" {
		version += " (commit " + commit + ")"
	}
	return version
}
