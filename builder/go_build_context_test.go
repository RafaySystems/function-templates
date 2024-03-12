package builder

import (
	"fmt"
	"io/fs"
	"testing"

	"github.com/RafaySystems/function-templates/builder/fixturesfs"
)

func TestGoBuild(t *testing.T) {
	err := fs.WalkDir(fixturesfs.FS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fmt.Println("path::", path)
		if !d.IsDir() {
			b, err := fs.ReadFile(fixturesfs.FS, path)
			if err != nil {
				return err
			}
			fmt.Println("file::", string(b))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

}
