module github.com/open-cluster-management/cluster-proxy-addon

go 1.16

require (
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/open-cluster-management/addon-framework v0.0.0-20210621074027-a81f712c10c2
	github.com/open-cluster-management/api v0.0.0-20210527013639-a6845f2ebcb1
	github.com/openshift/build-machinery-go v0.0.0-20210423112049-9415d7ebd33e
	github.com/openshift/library-go v0.0.0-20210609150209-1c980926414c
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	k8s.io/component-base v0.21.3
	k8s.io/klog/v2 v2.8.0
	open-cluster-management.io/addon-framework v0.0.0-20210709073210-719dbb79d275
	open-cluster-management.io/api v0.0.0-20210727123024-41c7397e9f2d
	open-cluster-management.io/registration-operator v0.4.0
	sigs.k8s.io/controller-runtime v0.9.5
)

replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.1 // ensure compatible between controller-runtime and kube-openapi
	open-cluster-management.io/registration-operator v0.4.0 => github.com/open-cluster-management-io/registration-operator v0.4.0
)
