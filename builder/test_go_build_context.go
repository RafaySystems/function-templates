package builder

import (
	"io/fs"
	"testing"

	go_templates "github.com/RafaySystems/function-templates/templates/go"
)

func TestGoBuildContextGetter(t *testing.T) {
	templateFS := go_templates.FS
	fs.WalkDir(templateFS, ".", func(path string, d fs.DirEntry, err error) error {
		t.Logf("path: %s", path)
		return nil
	})
}
