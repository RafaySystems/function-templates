package builder

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

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

	functionHandlerTemplateStr = "handler.{{.language}}.tmpl"
)

var (
	functionHandlerTemplate = template.Must(template.New("").Parse(functionHandlerTemplateStr))
)

type buildContextGetter struct {
	tmpdir          string
	fixtures        fs.FS
	templatesFolder string
}

/*
How to prepare the build context for a $language function
1. create temp directory
2. copy the function template from fixturesfs templates/$language to the temp directory
3. render the templates using the function data
5. tar the temp directory
6. write the tar to the writer
*/

func (b *buildContextGetter) GetBuildContext(options BuildContextGetterOptions, writer io.Writer) error {

	buildPath, err := os.MkdirTemp(b.tmpdir, "build-context-*")
	if err != nil {
		return err
	}

	//defer os.RemoveAll(buildPath)

	srcPath := filepath.Join(b.templatesFolder, options.Language)

	// copy the function template from fixturesfs templates/$language to the temp directory
	err = copy.Copy(srcPath, buildPath, copy.Options{
		FS:                b.fixtures,
		PermissionControl: copy.AddPermission(0777),
	})
	if err != nil {
		return err
	}

	// render the templates using the function data
	parsed, err := template.ParseFS(b.fixtures, "templates/**/function/*.tmpl")
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)

	data := map[string]interface{}{
		"language": options.Language,
		"source":   options.Source,
		"imports":  options.Imports,
	}

	if err = functionHandlerTemplate.Execute(buf, data); err != nil {
		return err
	}

	functionTemplatePath := buf.String()

	buf.Reset()

	if err := parsed.ExecuteTemplate(buf, functionTemplatePath, data); err != nil {
		return err
	}

	if err = os.WriteFile(filepath.Join(buildPath, "function/handler."+options.Language), buf.Bytes(), 0644); err != nil {
		return err
	}

	return nil
}
