module open-cluster-management.io/cluster-proxy-addon

go 1.16

require (
	github.com/cloudflare/cfssl v1.6.0
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/openshift/build-machinery-go v0.0.0-20210712174854-1bb7fd1518d3
	github.com/openshift/library-go v0.0.0-20210803154958-0e70d0844e00
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.22.0-rc.0
	k8s.io/apimachinery v0.22.0-rc.0
	k8s.io/client-go v0.22.0-rc.0
	k8s.io/component-base v0.22.0-rc.0
	k8s.io/klog/v2 v2.9.0
	open-cluster-management.io/addon-framework v0.0.0-20210803032803-58eac513499e
	open-cluster-management.io/api v0.0.0-20210727123024-41c7397e9f2d
	sigs.k8s.io/controller-runtime v0.9.5
)
