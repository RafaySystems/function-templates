package fixturesfs

import "io/fs"

var assertDir = func(path string) ([]string, error) {
	if path == "." {
		var children []string
		for c := range _bintree.Children {
			children = append(children, c)
		}
		return children, nil
	}
	return AssetDir(path)
}

func assetFS() *AssetFS {
	return &AssetFS{Asset: Asset, AssetDir: assertDir, AssetInfo: AssetInfo}
	// for k := range _bintree.Children {
	// 	return &AssetFS{Asset: Asset, AssetDir: assertDir, AssetInfo: AssetInfo, Prefix: k}
	// }
	// panic("unreachable")
}

var FS fs.FS = assetFS()
