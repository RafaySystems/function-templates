package builder

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/RafaySystems/function-templates/builder/fixturesfs"
)

func TestBuildContextLanguageGo(t *testing.T) {

	language := "go"

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile("testdata/" + language + "/handler.source")
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)

	bc := &buildContextGetter{
		tmpdir:          filepath.Join(wd, "testdata"),
		fixtures:        fixturesfs.FS,
		templatesFolder: TemplatesFolder,
	}
	err = bc.GetBuildContext(context.TODO(), BuildContextGetterOptions{
		Language: language,
		Source:   string(b),
		SourceDependencies: []string{
			"entgo.io/ent v0.12.5",
		},
	}, buf)
	if err != nil {
		t.Fatal(err)
	}

	writePath := "testdata/" + language + "-build.tar.gz"

	err = os.WriteFile(writePath, buf.Bytes(), 0644)
	if err != nil {
		t.Fatal(err)
	}

	defer os.Remove(writePath)
}

func TestBuildContextLanguagePython(t *testing.T) {

	language := "python"

	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile("testdata/" + language + "/handler.source")
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)

	bc := &buildContextGetter{
		tmpdir:          filepath.Join(wd, "testdata"),
		fixtures:        fixturesfs.FS,
		templatesFolder: TemplatesFolder,
	}
	err = bc.GetBuildContext(context.TODO(), BuildContextGetterOptions{
		Language: language,
		Source:   string(b),
	}, buf)
	if err != nil {
		t.Fatal(err)
	}
}
