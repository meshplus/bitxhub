package repo

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/gobuffalo/packd"
	"github.com/gobuffalo/packr"
	"github.com/meshplus/bitxhub-kit/fileutil"
)

const (
	packPath = "../../config"
)

func Initialize(repoRoot string) error {
	box := packr.NewBox(packPath)
	if err := box.Walk(func(s string, file packd.File) error {
		p := filepath.Join(repoRoot, s)
		dir := filepath.Dir(p)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				return fmt.Errorf("create folder failed: %w", err)
			}
		}

		return ioutil.WriteFile(p, []byte(file.String()), 0644)
	}); err != nil {
		return fmt.Errorf("traverse failed: %w", err)
	}

	return nil
}

func Initialized(repoRoot string) bool {
	return fileutil.Exist(filepath.Join(repoRoot, configName))
}
