package bitxhub

import (
	"fmt"
	"runtime"
)

var (
	// CurrentCommit current git commit hash
	CurrentCommit = ""
	// CurrentBranch current git branch
	CurrentBranch = ""
	// CurrentVersion current project version
	CurrentVersion = "0.0.0"
	// BuildDate compile date
	BuildDate = ""
	// GoVersion system go version
	GoVersion = runtime.Version()
	// Platform info
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)
