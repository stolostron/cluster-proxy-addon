// Code generated for package bindata by go-bindata DO NOT EDIT. (@generated)
// sources:
// deploy/managedcluster/manifest/addon_deployment.yaml
// deploy/managedcluster/manifest/anp_deployment.yaml
// deploy/managedcluster/manifest/clusterRoleBinding.yaml
// deploy/managedcluster/manifest/serviceaccount.yaml
package bindata

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func bindataRead(data []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewBuffer(data))
	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}

	var buf bytes.Buffer
	_, err = io.Copy(&buf, gz)
	clErr := gz.Close()

	if err != nil {
		return nil, fmt.Errorf("Read %q: %v", name, err)
	}
	if clErr != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _deployManagedclusterManifestAddon_deploymentYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\x52\x4d\xab\xdb\x30\x10\xbc\xfb\x57\x2c\xef\xae\x9a\x77\x15\xbc\x43\x48\x2f\xa1\x6d\x30\x14\x7a\x5f\xcb\x1b\x5b\x44\x96\x84\x3e\x4c\x43\xf0\x7f\x2f\xb2\xec\xc4\x6e\x3e\xf6\x24\x76\x3c\x33\xbb\xb3\x3e\x4b\xdd\x70\xf8\x4e\x56\x99\x4b\x4f\x3a\x14\x68\xe5\x1f\x72\x5e\x1a\xcd\x01\xad\xf5\xe5\xf0\x59\xf4\x14\xb0\xc1\x80\xbc\x00\xd0\xd8\x13\x07\xa1\xa2\x0f\xe4\x98\x75\xe6\xef\x85\x61\x9b\x98\x19\xf3\x16\x05\x71\xb8\x5e\xe1\xdb\xae\x69\x8c\x3e\x68\x1f\x50\xa9\xe3\x02\xc1\x38\x16\x00\x0a\x6b\x52\x3e\xe9\x41\x72\x79\x2e\xe8\x2d\x89\xf4\x89\x23\xab\xa4\x40\xcf\xe1\xb3\x00\xf0\xa4\x48\x04\xe3\x32\xb9\xc7\x20\xba\x9f\x2b\xb5\x37\x7a\x00\x81\x7a\xab\x30\xd0\xcc\x5d\xad\x95\x4a\x6d\x64\xde\x0a\x01\x2c\xc3\x4d\x6f\x72\x83\x14\xb4\x13\xc2\x44\x1d\x8e\xaf\x12\x62\x1e\x67\xc2\x60\x54\xec\x69\x65\xc5\xe6\x5c\xbb\x58\x33\x61\xf4\x49\xb6\x37\x28\xc9\x0b\x47\x81\xaf\x3a\x4b\x2f\x5b\xa5\xac\x7f\xc4\x9a\xf6\x13\xf1\xf7\x84\xe4\x98\x53\x09\xa3\x03\x4a\x4d\xee\x89\xdd\xab\xe5\x72\xc9\x1e\xdb\x59\xfe\x90\x9e\x77\xcd\x1b\x5a\x45\xa5\x2a\xa3\xa4\xb8\x70\x38\x9c\x8e\x26\x54\x8e\xfc\x56\x05\x5d\xeb\xb7\xb3\x33\xf8\x28\x37\xce\x1f\xff\xc3\xd3\x24\x0f\x5d\xc6\x52\x3c\xe7\x58\x53\x8e\xe8\xab\x1c\xd0\x95\x2e\xea\xb2\x8b\x75\x79\xef\x3f\x21\x2e\x76\x69\xed\xaf\xb4\xd0\x3e\x37\x52\x80\x30\x8e\x6b\x46\xbe\xcd\xaf\x74\xc9\x87\xb9\xdf\x1c\x69\xfa\xa1\x12\xa9\xc2\xd0\x71\x58\x8f\x56\xfc\x0b\x00\x00\xff\xff\xff\xf6\xa6\xa5\x65\x03\x00\x00")

func deployManagedclusterManifestAddon_deploymentYamlBytes() ([]byte, error) {
	return bindataRead(
		_deployManagedclusterManifestAddon_deploymentYaml,
		"deploy/managedcluster/manifest/addon_deployment.yaml",
	)
}

func deployManagedclusterManifestAddon_deploymentYaml() (*asset, error) {
	bytes, err := deployManagedclusterManifestAddon_deploymentYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "deploy/managedcluster/manifest/addon_deployment.yaml", size: 869, mode: os.FileMode(420), modTime: time.Unix(1628061261, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _deployManagedclusterManifestAnp_deploymentYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\xa4\x53\x4d\x6b\xdb\x40\x10\xbd\xfb\x57\x0c\xbe\xcb\x8a\x5b\x4a\x5b\x81\x0f\xa1\x81\x26\xd0\xa6\x82\x84\xde\x27\xab\x69\xb4\x78\xbf\xd8\x1d\x99\xaa\xc1\xff\xbd\xac\x56\x76\x57\xb2\x92\x4b\x75\x30\xcb\xbc\x79\x6f\xde\x7c\x18\x9d\xfc\x49\x3e\x48\x6b\x2a\x40\xe7\x42\x79\xd8\xae\xf6\xd2\x34\x15\xdc\x90\x53\xb6\xd7\x64\x78\xa5\x89\xb1\x41\xc6\x6a\x05\xa0\xf0\x89\x54\x88\x2f\x88\x84\x0a\xd0\xb8\x02\x9f\x63\x1a\x80\x41\x4d\x79\x24\x38\x12\x31\x35\x90\x22\xc1\xd6\x27\x9a\x46\x16\xed\xb7\x4c\x67\x41\x89\x49\x3b\x85\x4c\x23\x23\x33\x10\x3f\x35\x21\x2f\xd0\x01\x4e\xa5\xe3\xd7\xda\xc0\xd7\x4a\x62\xa0\x8c\x53\x80\x74\x15\xac\xb7\xef\x3e\x6e\xae\x36\x57\x9b\xed\xfa\x8c\x24\x42\xec\x25\x4b\x4f\x94\xf5\xcb\x0b\x6c\xbe\xa8\x2e\x30\xf9\x7b\xd4\x04\xc7\xe3\x89\x27\xac\x61\x94\x86\xfc\xa4\xc6\x38\x11\x27\x03\xf9\x03\xf9\xc2\x79\xfb\xbb\xcf\x44\xa5\xc6\x67\xaa\x20\xca\xde\xc5\x27\x1c\x8f\x19\x2a\xac\xd6\x68\x9a\x0b\x17\xe5\x4c\x30\xf7\xee\xac\xe7\x0b\xdf\x67\x73\xb5\xf5\x9c\xca\x5d\xd7\x77\x0f\x83\x44\x1d\x15\x62\x3c\xaf\x7d\x72\x3e\xc8\x67\x63\xfd\x2f\xd7\x99\xd8\x7a\x0e\x16\xa9\x93\x62\x6c\x2b\x6e\x60\x17\xf5\x07\x77\xc9\xe8\xad\x0d\x9c\x0d\xfc\x35\x6a\x1c\xc0\x9c\x3a\xb6\xb7\x40\x1d\xdc\x14\xb2\x21\xc3\xf2\x97\x24\x1f\x76\xe7\xd2\xcb\x7b\xce\xb8\x02\x0b\x41\x9e\x77\x65\xfc\x0d\xa5\x48\xf9\xa3\x1b\x81\x1b\xe1\x97\xfa\x4c\x15\x5f\x27\x0e\xf8\x9b\xdc\x3d\xf5\x6f\x51\xf7\x34\x39\x08\x25\x0f\x64\x28\x84\xda\xdb\x27\x9a\x2e\xa5\x65\x76\x5f\x89\xa7\x41\x80\x20\x5a\x8a\xcb\xbf\x7d\x7c\xac\x67\x90\x1b\xee\xe7\xd3\xd5\xe7\xf7\x73\x00\xb9\xad\xa0\x6c\x09\x15\xb7\x7f\x26\xa0\x34\x92\x25\xaa\x1b\x52\xd8\x3f\x90\xb0\xa6\x09\x15\x6c\x3f\x4c\x72\x58\x6a\xb2\x1d\x2f\xc3\x07\xab\x3a\x4d\xdf\x6d\x67\x2e\x2f\x3b\x9d\xe9\x30\x8c\x99\x23\x1d\xf3\xeb\x64\x6b\x09\xf7\x84\xcd\x0f\xa3\xfa\x0a\xd8\x77\xb4\xca\x4b\x2d\xfc\x85\x2f\x24\x02\x09\x3f\x1f\x5d\x8a\xdd\xff\xb3\xf4\x37\x00\x00\xff\xff\x04\x8f\xe3\x22\x5f\x05\x00\x00")

func deployManagedclusterManifestAnp_deploymentYamlBytes() ([]byte, error) {
	return bindataRead(
		_deployManagedclusterManifestAnp_deploymentYaml,
		"deploy/managedcluster/manifest/anp_deployment.yaml",
	)
}

func deployManagedclusterManifestAnp_deploymentYaml() (*asset, error) {
	bytes, err := deployManagedclusterManifestAnp_deploymentYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "deploy/managedcluster/manifest/anp_deployment.yaml", size: 1375, mode: os.FileMode(420), modTime: time.Unix(1628065717, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _deployManagedclusterManifestClusterrolebindingYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x7c\xce\x31\x4e\x43\x31\x0c\xc6\xf1\x3d\xa7\xf0\x05\xde\x43\x6c\x28\x5b\x61\x40\x2c\x0c\x45\x62\x77\x13\x53\x4c\xf3\xec\xc8\x76\x2a\xa0\xea\xdd\x51\x05\x74\xa9\xd4\xd9\x9f\x7f\xfa\xef\x58\x6a\x86\x87\x36\x3c\xc8\xd6\xda\xe8\x9e\xa5\xb2\x6c\x13\x76\x7e\x25\x73\x56\xc9\x60\x1b\x2c\x33\x8e\x78\x57\xe3\x6f\x0c\x56\x99\x77\x77\x3e\xb3\xde\xec\x6f\xd3\x42\x81\x15\x03\x73\x02\x10\x5c\x28\x43\xf9\xd5\xa6\x6e\xfa\xf9\x35\xe1\x96\x24\x92\x69\xa3\x35\xbd\x9d\x46\xd8\xf9\xd1\x74\xf4\x2b\x6e\x02\xb8\x08\xbb\xe0\xb1\x2e\x2c\xc9\xc7\xe6\x83\x4a\xf8\x49\x9e\xfe\xbe\x5e\xc8\xf6\x5c\x68\x55\x8a\x0e\x89\x04\x70\xa5\x6c\x72\x3c\x0f\xbc\x63\xa1\x0c\x87\x03\xcc\xab\x5a\x55\x9e\xc4\x03\x5b\x7b\xfe\x3f\xc1\xf1\x98\x7e\x02\x00\x00\xff\xff\x1a\x1f\x33\x7e\x31\x01\x00\x00")

func deployManagedclusterManifestClusterrolebindingYamlBytes() ([]byte, error) {
	return bindataRead(
		_deployManagedclusterManifestClusterrolebindingYaml,
		"deploy/managedcluster/manifest/clusterRoleBinding.yaml",
	)
}

func deployManagedclusterManifestClusterrolebindingYaml() (*asset, error) {
	bytes, err := deployManagedclusterManifestClusterrolebindingYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "deploy/managedcluster/manifest/clusterRoleBinding.yaml", size: 305, mode: os.FileMode(420), modTime: time.Unix(1628061121, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _deployManagedclusterManifestServiceaccountYaml = []byte("\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\xff\x34\xc9\x31\xaa\xc3\x30\x0c\x06\xe0\xdd\xa7\xd0\x05\xfc\xe0\xad\xda\x32\x76\xe9\x52\xe8\xfe\x23\x8b\x62\xea\xc8\xc6\x52\x42\x4b\xc8\xdd\xbb\xb4\xf3\xf7\xac\x56\x98\x6e\x3a\xf7\x2a\xba\x88\xf4\xcd\x22\x61\xd4\xbb\x4e\xaf\xdd\x98\xf6\xff\xb4\x6a\xa0\x20\xc0\x89\xc8\xb0\x2a\x93\xb4\xcd\x43\x67\x1e\xb3\xbf\xde\x19\x0f\xb5\xc8\x8e\x2f\xfb\x80\x28\xd3\x71\xd0\xdf\x52\x4a\xb7\x8b\x79\xa0\xb5\xeb\x8f\xe8\x3c\xd3\x27\x00\x00\xff\xff\x8a\xaf\x1e\x2c\x77\x00\x00\x00")

func deployManagedclusterManifestServiceaccountYamlBytes() ([]byte, error) {
	return bindataRead(
		_deployManagedclusterManifestServiceaccountYaml,
		"deploy/managedcluster/manifest/serviceaccount.yaml",
	)
}

func deployManagedclusterManifestServiceaccountYaml() (*asset, error) {
	bytes, err := deployManagedclusterManifestServiceaccountYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "deploy/managedcluster/manifest/serviceaccount.yaml", size: 119, mode: os.FileMode(420), modTime: time.Unix(1628061106, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"deploy/managedcluster/manifest/addon_deployment.yaml":   deployManagedclusterManifestAddon_deploymentYaml,
	"deploy/managedcluster/manifest/anp_deployment.yaml":     deployManagedclusterManifestAnp_deploymentYaml,
	"deploy/managedcluster/manifest/clusterRoleBinding.yaml": deployManagedclusterManifestClusterrolebindingYaml,
	"deploy/managedcluster/manifest/serviceaccount.yaml":     deployManagedclusterManifestServiceaccountYaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"deploy": &bintree{nil, map[string]*bintree{
		"managedcluster": &bintree{nil, map[string]*bintree{
			"manifest": &bintree{nil, map[string]*bintree{
				"addon_deployment.yaml":   &bintree{deployManagedclusterManifestAddon_deploymentYaml, map[string]*bintree{}},
				"anp_deployment.yaml":     &bintree{deployManagedclusterManifestAnp_deploymentYaml, map[string]*bintree{}},
				"clusterRoleBinding.yaml": &bintree{deployManagedclusterManifestClusterrolebindingYaml, map[string]*bintree{}},
				"serviceaccount.yaml":     &bintree{deployManagedclusterManifestServiceaccountYaml, map[string]*bintree{}},
			}},
		}},
	}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
