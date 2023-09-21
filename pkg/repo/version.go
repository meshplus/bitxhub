package repo

import (
	"fmt"
	"runtime"
)

var (
	// BuildCommit build git commit hash
	BuildCommit = ""

	// BuildBranch build git branch
	BuildBranch = ""

	// BuildVersion build project version
	BuildVersion = "0.0.0"

	// BuildDate compile date
	BuildDate = ""

	BuildVersionSecret = "Hangzhou"

	BuildNet = ""

	// GoVersion system go version
	GoVersion = runtime.Version()

	// Platform info
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)
