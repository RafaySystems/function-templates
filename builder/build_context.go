package builder

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/mholt/archives"
	"github.com/otiai10/copy"
)

const (
	// GoLanguage is the language for go
	GoLanguage = "go"
	// PythonLanguage is the language for python
	PythonLanguage = "python"
	// BashLanguage is the language for bash
	BashLanguage = "bash"

	// TemplatesFolder is the folder for the templates in the fixtures
	TemplatesFolder = "templates"
)

var (
	sourceInfo = map[string]struct {
		HandlerExtension string
		TemplatePath     string
		DestPath         string
	}{
		GoLanguage:     {"go", "function/go.mod.tmpl", "function/go.mod"},
		PythonLanguage: {"py", "function/requirements.txt.tmpl", "function/requirements.txt"},
	}
)

type buildContextGetter struct {
	tmpdir          string
	fixtures        fs.FS
	templatesFolder string
}

func NewBuildContextGetter(tmpdir string, fixtures fs.FS) *buildContextGetter {
	return &buildContextGetter{
		tmpdir:          tmpdir,
		fixtures:        fixtures,
		templatesFolder: TemplatesFolder,
	}
}

/*
How to prepare the build context for a $language function
1. create temp directory
2. copy the function template from fixturesfs templates/$language to the temp directory
3. render the templates using the function data
5. tar the temp directory
6. write the tar to the writer
*/

func (b *buildContextGetter) GetBuildContext(ctx context.Context, options BuildContextGetterOptions, writer io.Writer) error {

	buildPath, err := os.MkdirTemp(b.tmpdir, "build-context-*")
	if err != nil {
		return err
	}

	defer os.RemoveAll(buildPath)

	srcPath := filepath.Join(b.templatesFolder, options.Language)

	// copy the function template from fixturesfs templates/$language to the temp directory
	err = copy.Copy(srcPath, buildPath, copy.Options{
		FS:                b.fixtures,
		PermissionControl: copy.AddPermission(0777),
	})
	if err != nil {
		return err
	}

	sourceDependenciesTemplate, err := template.ParseFS(b.fixtures, fmt.Sprintf("templates/%s/%s", options.Language, sourceInfo[options.Language].TemplatePath))
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)

	data := map[string]interface{}{
		"source_dependencies": options.SourceDependencies,
	}

	// render the templates using the function data
	err = sourceDependenciesTemplate.Execute(buf, data)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(filepath.Join(buildPath, sourceInfo[options.Language].DestPath), os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	_, err = io.Copy(f, buf)
	if err != nil {
		return err
	}

	if err = os.WriteFile(filepath.Join(buildPath, "function/handler."+sourceInfo[options.Language].HandlerExtension), []byte(options.Source), 0644); err != nil {
		return err
	}

	tarFiles := map[string]string{}

	err = filepath.WalkDir(buildPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		tarFiles[path] = strings.TrimPrefix(path, buildPath)
		return nil
	})
	if err != nil {
		return err
	}

	files, err := archives.FilesFromDisk(ctx, &archives.FromDiskOptions{}, tarFiles)
	if err != nil {
		return err
	}

	format := archives.CompressedArchive{
		Compression: archives.Gz{},
		Archival:    archives.Tar{},
	}

	err = format.Archive(ctx, writer, files)
	if err != nil {
		return err
	}

	return nil
}
