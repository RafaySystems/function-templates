package fixturesfs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// Adapted from https://github.com/elazarl/go-bindata-assetfs
// Original license: BSD-2-Clause license

var (
	defaultFileTimestamp = time.Now()
)

// FakeFile implements os.FileInfo interface for a given path and size
type FakeFile struct {
	// Path is the path of this file
	Path string
	// Dir marks of the path is a directory
	Dir bool
	// Len is the length of the fake file, zero if it is a directory
	Len int64
	// Timestamp is the ModTime of this file
	Timestamp time.Time
}

func (f *FakeFile) Name() string {
	_, name := filepath.Split(f.Path)
	return name
}

func (f *FakeFile) Mode() os.FileMode {
	mode := os.FileMode(0644)
	if f.Dir {
		return mode | os.ModeDir
	}
	return mode
}

func (f *FakeFile) ModTime() time.Time {
	return f.Timestamp
}

func (f *FakeFile) Size() int64 {
	return f.Len
}

func (f *FakeFile) IsDir() bool {
	return f.Mode().IsDir()
}

func (f *FakeFile) Sys() interface{} {
	return nil
}

func (f *FakeFile) Info() (fs.FileInfo, error) {
	return f, nil
}

func (f *FakeFile) Type() fs.FileMode {
	if f.Dir {
		return fs.ModeDir
	}
	return 0
}

var _ fs.DirEntry = (*FakeFile)(nil)

// AssetFile implements http.File interface for a no-directory file with content
type AssetFile struct {
	*bytes.Reader
	io.Closer
	FakeFile
}

func NewAssetFile(name string, content []byte, timestamp time.Time) *AssetFile {
	if timestamp.IsZero() {
		timestamp = defaultFileTimestamp
	}
	return &AssetFile{
		bytes.NewReader(content),
		io.NopCloser(nil),
		FakeFile{name, false, int64(len(content)), timestamp}}
}

func (f *AssetFile) Readdir(count int) ([]os.FileInfo, error) {
	return nil, errors.New("not a directory")
}

func (f *AssetFile) Size() int64 {
	return f.FakeFile.Size()
}

func (f *AssetFile) Stat() (os.FileInfo, error) {
	return f, nil
}

func (f *AssetFile) Info() (fs.FileInfo, error) {
	return f, nil
}

// AssetDirectory implements http.File interface for a directory
type AssetDirectory struct {
	AssetFile
	ChildrenRead int
	Children     []fs.DirEntry
}

func NewAssetDirectory(name string, children []string, afs *AssetFS) *AssetDirectory {
	fileinfos := make([]fs.DirEntry, 0, len(children))
	for _, child := range children {
		_, err := afs.AssetDir(filepath.Join(name, child))
		fileinfos = append(fileinfos, &FakeFile{child, err == nil, 0, time.Time{}})
	}
	return &AssetDirectory{
		AssetFile{
			bytes.NewReader(nil),
			io.NopCloser(nil),
			FakeFile{name, true, 0, time.Time{}},
		},
		0,
		fileinfos}
}

var _ fs.ReadDirFile = (*AssetDirectory)(nil)

func (f *AssetDirectory) ReadDir(count int) ([]os.DirEntry, error) {
	if count <= 0 {
		return f.Children, nil
	}
	if f.ChildrenRead+count > len(f.Children) {
		count = len(f.Children) - f.ChildrenRead
	}
	rv := f.Children[f.ChildrenRead : f.ChildrenRead+count]
	f.ChildrenRead += count
	return rv, nil
}

func (f *AssetDirectory) Stat() (os.FileInfo, error) {
	return f, nil
}

// AssetFS implements http.FileSystem, allowing
// embedded files to be served from net/http package.
type AssetFS struct {
	// Asset should return content of file in path if exists
	Asset func(path string) ([]byte, error)
	// AssetDir should return list of files in the path
	AssetDir func(path string) ([]string, error)
	// AssetInfo should return the info of file in path if exists
	AssetInfo func(path string) (os.FileInfo, error)
	// Prefix would be prepended to http requests
	Prefix string
	// Fallback file that is served if no other is found
	Fallback string
}

var _ fs.FS = (*AssetFS)(nil)

func (fs *AssetFS) Open(name string) (fs.File, error) {
	name = path.Join(fs.Prefix, name)
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}
	if b, err := fs.Asset(name); err == nil {
		timestamp := defaultFileTimestamp
		if fs.AssetInfo != nil {
			if info, err := fs.AssetInfo(name); err == nil {
				timestamp = info.ModTime()
			}
		}
		return NewAssetFile(name, b, timestamp), nil
	}
	children, err := fs.AssetDir(name)

	if err != nil {
		if len(fs.Fallback) > 0 {
			return fs.Open(fs.Fallback)
		}

		// If the error is not found, return an error that will
		// result in a 404 error. Otherwise the server returns
		// a 500 error for files not found.
		if strings.Contains(err.Error(), "not found") {
			return nil, os.ErrNotExist
		}
		return nil, err
	}

	return NewAssetDirectory(name, children, fs), nil
}
