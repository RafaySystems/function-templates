package fixturesfs

import (
	"io/fs"
	"testing"
)

func TestAssetFS(t *testing.T) {
	err := fs.WalkDir(FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			_, err := fs.ReadFile(FS, path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

}
