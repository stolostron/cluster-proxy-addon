// Code generated for package bindata by go-bindata DO NOT EDIT. (@generated)
// sources:
// pkg/hub/addon/manifests/addon_deployment.yaml
// pkg/hub/addon/manifests/anp_deployment.yaml
// pkg/hub/addon/manifests/clusterrole.yaml
// pkg/hub/addon/manifests/clusterrolebinding.yaml
// pkg/hub/addon/manifests/configmap.yaml
// pkg/hub/addon/manifests/namespace.yaml
// pkg/hub/addon/manifests/role.yaml
// pkg/hub/addon/manifests/rolebinding.yaml
// pkg/hub/addon/manifests/serviceaccount.yaml
package bindata

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

var _pkgHubAddonManifestsAddon_deploymentYaml = []byte(`kind: Deployment
apiVersion: apps/v1
metadata:
  name: cluster-proxy-agent
  namespace: {{ .AddonInstallNamespace }}
  labels:
    app: cluster-proxy-agent
spec:
  replicas: 1
  selector:
    matchLabels:
      app: cluster-proxy-agent
  template:
    metadata:
      labels:
        app: cluster-proxy-agent
    spec:
      serviceAccountName: cluster-proxy-agent-sa
      volumes:
        - name: hub-config
          secret:
            secretName: {{ .KubeConfigSecret }}
      containers:
        - name: cluster-proxy-agent
          image: {{ .Image }}
          args:
            - "/cluster-proxy"
            - "agent"
            - "--hub-kubeconfig=/var/run/hub/kubeconfig"
            - "--cluster-name={{ .ClusterName }}"
          volumeMounts:
            - name: hub-config
              mountPath: /var/run/hub
`)

func pkgHubAddonManifestsAddon_deploymentYamlBytes() ([]byte, error) {
	return _pkgHubAddonManifestsAddon_deploymentYaml, nil
}

func pkgHubAddonManifestsAddon_deploymentYaml() (*asset, error) {
	bytes, err := pkgHubAddonManifestsAddon_deploymentYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "pkg/hub/addon/manifests/addon_deployment.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _pkgHubAddonManifestsAnp_deploymentYaml = []byte(`kind: Deployment
apiVersion: apps/v1
metadata:
  name: anp-agent
  namespace: {{ .AddonInstallNamespace }}
  labels:
    app: anp-agent
spec:
  selector:
    matchLabels:
      app: anp-agent
  template:
    metadata:
      labels:
        app: anp-agent
    spec:
      hostAliases:
        - ip: "127.0.0.1"
          hostnames:
            - "{{ .ClusterName }}"
      containers:
        - name: apiserver-proxy
          image: {{ .Image }}
          command:
            - "/cluster-proxy"
            - "apiserver-proxy"
          ports:
            - containerPort: {{ .APIServerProxyPort }}
        - name: proxy-agent-config-checker
          image: {{ .Image }}
          command:
            - "/cluster-proxy"
            - "config-checker"
            - "--name=proxy-agent-config-checker"
            - "--port=8180"
            - "--files=/ca/ca-bundle.crt"
            - "--files=/tls/tls.crt"
            - "--files=/tls/tls.key"
          volumeMounts:
            - name: tls-vol
              mountPath: /tls
              readOnly: true
            - name: ca-vol
              mountPath: /ca
              readOnly: true
        - name: proxy-agent
          image: {{ .Image }}
          command:
            - "/proxy-agent"
            - "--proxy-server-host={{ .ProxyServerHost }}"
            - "--proxy-server-port={{ .ProxyServerPort }}"
            - "--agent-identifiers=host={{ .ClusterName }}"
            - "--ca-cert=/ca/ca-bundle.crt"
            - "--agent-cert=/tls/tls.crt"
            - "--agent-key=/tls/tls.key"
          livenessProbe:
            httpGet:
              scheme: HTTP
              port: 8180
              path: "/proxy-agent-config-checker"
            initialDelaySeconds: 60
            timeoutSeconds: 60
            failureThreshold: 1
          volumeMounts:
            - name: tls-vol
              mountPath: /tls
              readOnly: true
            - name: ca-vol
              mountPath: /ca
              readOnly: true
      volumes:
        - name: tls-vol
          secret:
            secretName: cluster-proxy-open-cluster-management.io-cluster-proxy-signer-client-cert
        - name: ca-vol
          configMap:
            name: cluster-proxy-ca-bundle
`)

func pkgHubAddonManifestsAnp_deploymentYamlBytes() ([]byte, error) {
	return _pkgHubAddonManifestsAnp_deploymentYaml, nil
}

func pkgHubAddonManifestsAnp_deploymentYaml() (*asset, error) {
	bytes, err := pkgHubAddonManifestsAnp_deploymentYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "pkg/hub/addon/manifests/anp_deployment.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _pkgHubAddonManifestsClusterroleYaml = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: open-cluster-management:cluster-proxy-addon:agent
rules:
# Allow cluster-proxy-addon agent to run with openshift library-go
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch"]

`)

func pkgHubAddonManifestsClusterroleYamlBytes() ([]byte, error) {
	return _pkgHubAddonManifestsClusterroleYaml, nil
}

func pkgHubAddonManifestsClusterroleYaml() (*asset, error) {
	bytes, err := pkgHubAddonManifestsClusterroleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "pkg/hub/addon/manifests/clusterrole.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _pkgHubAddonManifestsClusterrolebindingYaml = []byte(`kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: cluster-proxy-agent
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: open-cluster-management:cluster-proxy-addon:agent
subjects:
  - kind: ServiceAccount
    name: cluster-proxy-agent-sa
    namespace: {{ .AddonInstallNamespace }}
`)

func pkgHubAddonManifestsClusterrolebindingYamlBytes() ([]byte, error) {
	return _pkgHubAddonManifestsClusterrolebindingYaml, nil
}

func pkgHubAddonManifestsClusterrolebindingYaml() (*asset, error) {
	bytes, err := pkgHubAddonManifestsClusterrolebindingYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "pkg/hub/addon/manifests/clusterrolebinding.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _pkgHubAddonManifestsConfigmapYaml = []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-proxy-ca-bundle
  namespace: {{ .AddonInstallNamespace }}
data:
  ca-bundle.crt: | 
    {{ .CABundleCrt | indent 4 }}
`)

func pkgHubAddonManifestsConfigmapYamlBytes() ([]byte, error) {
	return _pkgHubAddonManifestsConfigmapYaml, nil
}

func pkgHubAddonManifestsConfigmapYaml() (*asset, error) {
	bytes, err := pkgHubAddonManifestsConfigmapYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "pkg/hub/addon/manifests/configmap.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _pkgHubAddonManifestsNamespaceYaml = []byte(`apiVersion: v1
kind: Namespace
metadata:
  name: {{ .AddonInstallNamespace }}
`)

func pkgHubAddonManifestsNamespaceYamlBytes() ([]byte, error) {
	return _pkgHubAddonManifestsNamespaceYaml, nil
}

func pkgHubAddonManifestsNamespaceYaml() (*asset, error) {
	bytes, err := pkgHubAddonManifestsNamespaceYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "pkg/hub/addon/manifests/namespace.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _pkgHubAddonManifestsRoleYaml = []byte(`kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: cluster-proxy-addon
  namespace: {{ .AddonInstallNamespace }}
rules:
# Allow cluster-proxy-addon agent to run with openshift library-go
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list", "watch", "create", "delete", "update", "patch"]
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list"]
- apiGroups: ["apps"]
  resources: ["replicasets"]
  verbs: ["get"]
- apiGroups: ["", "events.k8s.io"]
  resources: ["events"]
  verbs: ["create", "patch", "update"]
# Allow cluster-proxy-addon agent to run with addon-framwork
- apiGroups: ["coordination.k8s.io"]
  resources: ["leases"]
  verbs: ["get", "list", "watch", "create", "update", "delete"]
`)

func pkgHubAddonManifestsRoleYamlBytes() ([]byte, error) {
	return _pkgHubAddonManifestsRoleYaml, nil
}

func pkgHubAddonManifestsRoleYaml() (*asset, error) {
	bytes, err := pkgHubAddonManifestsRoleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "pkg/hub/addon/manifests/role.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _pkgHubAddonManifestsRolebindingYaml = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: cluster-proxy-addon
  namespace: {{ .AddonInstallNamespace }}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: cluster-proxy-addon
subjects:
  - kind: ServiceAccount
    name: cluster-proxy-agent-sa
`)

func pkgHubAddonManifestsRolebindingYamlBytes() ([]byte, error) {
	return _pkgHubAddonManifestsRolebindingYaml, nil
}

func pkgHubAddonManifestsRolebindingYaml() (*asset, error) {
	bytes, err := pkgHubAddonManifestsRolebindingYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "pkg/hub/addon/manifests/rolebinding.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _pkgHubAddonManifestsServiceaccountYaml = []byte(`kind: ServiceAccount
apiVersion: v1
metadata:
  name: cluster-proxy-agent-sa
  namespace: {{ .AddonInstallNamespace }}
imagePullSecrets:
- name: open-cluster-management-image-pull-credentials
`)

func pkgHubAddonManifestsServiceaccountYamlBytes() ([]byte, error) {
	return _pkgHubAddonManifestsServiceaccountYaml, nil
}

func pkgHubAddonManifestsServiceaccountYaml() (*asset, error) {
	bytes, err := pkgHubAddonManifestsServiceaccountYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "pkg/hub/addon/manifests/serviceaccount.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
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
	"pkg/hub/addon/manifests/addon_deployment.yaml":   pkgHubAddonManifestsAddon_deploymentYaml,
	"pkg/hub/addon/manifests/anp_deployment.yaml":     pkgHubAddonManifestsAnp_deploymentYaml,
	"pkg/hub/addon/manifests/clusterrole.yaml":        pkgHubAddonManifestsClusterroleYaml,
	"pkg/hub/addon/manifests/clusterrolebinding.yaml": pkgHubAddonManifestsClusterrolebindingYaml,
	"pkg/hub/addon/manifests/configmap.yaml":          pkgHubAddonManifestsConfigmapYaml,
	"pkg/hub/addon/manifests/namespace.yaml":          pkgHubAddonManifestsNamespaceYaml,
	"pkg/hub/addon/manifests/role.yaml":               pkgHubAddonManifestsRoleYaml,
	"pkg/hub/addon/manifests/rolebinding.yaml":        pkgHubAddonManifestsRolebindingYaml,
	"pkg/hub/addon/manifests/serviceaccount.yaml":     pkgHubAddonManifestsServiceaccountYaml,
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
	"pkg": {nil, map[string]*bintree{
		"hub": {nil, map[string]*bintree{
			"addon": {nil, map[string]*bintree{
				"manifests": {nil, map[string]*bintree{
					"addon_deployment.yaml":   {pkgHubAddonManifestsAddon_deploymentYaml, map[string]*bintree{}},
					"anp_deployment.yaml":     {pkgHubAddonManifestsAnp_deploymentYaml, map[string]*bintree{}},
					"clusterrole.yaml":        {pkgHubAddonManifestsClusterroleYaml, map[string]*bintree{}},
					"clusterrolebinding.yaml": {pkgHubAddonManifestsClusterrolebindingYaml, map[string]*bintree{}},
					"configmap.yaml":          {pkgHubAddonManifestsConfigmapYaml, map[string]*bintree{}},
					"namespace.yaml":          {pkgHubAddonManifestsNamespaceYaml, map[string]*bintree{}},
					"role.yaml":               {pkgHubAddonManifestsRoleYaml, map[string]*bintree{}},
					"rolebinding.yaml":        {pkgHubAddonManifestsRolebindingYaml, map[string]*bintree{}},
					"serviceaccount.yaml":     {pkgHubAddonManifestsServiceaccountYaml, map[string]*bintree{}},
				}},
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
