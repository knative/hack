package docs

import (
	"go/build"
	"os"
	"path/filepath"
)

func gomodcache() string {
	// See: https://github.com/golang/go/blob/release-branch.go1.22/src/cmd/go/internal/cfg/cfg.go#L408
	return envOr("GOMODCACHE", gopathDir("pkg/mod"))
}

func gopathDir(rel string) string {
	list := filepath.SplitList(gopath())
	if len(list) == 0 || list[0] == "" {
		return ""
	}
	return filepath.Join(list[0], rel)
}

func gopath() string {
	gp := os.Getenv("GOPATH")
	if gp == "" {
		gp = build.Default.GOPATH
	}
	return gp
}

func envOr(key string, def string) string {
	val := os.Getenv(key)
	if val == "" {
		val = def
	}
	return val
}
