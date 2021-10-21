module github.com/open-cluster-management/cluster-proxy-addon

go 1.16

require (
	github.com/cloudflare/cfssl v1.6.0
	github.com/go-bindata/go-bindata v3.1.2+incompatible
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.16.0
	github.com/open-cluster-management/multicloud-operators-foundation v1.0.0-2021-09-15-06-57-52
	github.com/openshift/build-machinery-go v0.0.0-20210806203541-4ea9b6da3a37
	github.com/openshift/library-go v0.0.0-20210916194400-ae21aab32431
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/component-base v0.22.1
	k8s.io/klog/v2 v2.9.0
	open-cluster-management.io/addon-framework v0.0.0-20211020072210-50d435b65f83
	open-cluster-management.io/api v0.0.0-20210916013819-2e58cdb938f9
)

replace github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.5.1 // ensure compatible between controller-runtime and kube-openapi

replace k8s.io/apiserver => k8s.io/apiserver v0.22.1

// required by multicloud-operators-foundation
replace (
	github.com/kubevirt/terraform-provider-kubevirt => github.com/nirarg/terraform-provider-kubevirt v0.0.0-20201222125919-101cee051ed3
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20200715132148-0f91f62a41fe
	github.com/metal3-io/cluster-api-provider-baremetal => github.com/openshift/cluster-api-provider-baremetal v0.0.0-20190821174549-a2a477909c1d
	github.com/openshift/api => github.com/openshift/api v0.0.0-20210331193751-3acddb19d360
	github.com/openshift/hive/apis => github.com/openshift/hive/apis v0.0.0-20210802140536-4d8d83dcd464
	github.com/openshift/library-go => github.com/openshift/library-go v0.0.0-20210916194400-ae21aab32431
	github.com/terraform-providers/terraform-provider-ignition/v2 => github.com/community-terraform-providers/terraform-provider-ignition/v2 v2.1.0
	google.golang.org/grpc => google.golang.org/grpc v1.38.0
	k8s.io/client-go => k8s.io/client-go v0.22.1
	kubevirt.io/client-go => kubevirt.io/client-go v0.29.0
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20200506073438-9d49428ff837
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20200120114645-8a9592f1f87b
	sigs.k8s.io/cluster-api-provider-openstack => github.com/openshift/cluster-api-provider-openstack v0.0.0-20200526112135-319a35b2e38e
)
