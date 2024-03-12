package builder

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/RafaySystems/function-templates/builder/fixturesfs"
)

func TestBuildContext(t *testing.T) {

	language := "go"

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile("testdata/" + language + "/handler.source")
	if err != nil {
		t.Fatal(err)
	}

	bc := &buildContextGetter{
		tmpdir:          filepath.Join(wd, "testdata"),
		fixtures:        fixturesfs.FS,
		templatesFolder: TemplatesFolder,
	}
	err = bc.GetBuildContext(BuildContextGetterOptions{
		Language: "go",
		Source:   string(b),
		Imports:  []string{"time"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
}
