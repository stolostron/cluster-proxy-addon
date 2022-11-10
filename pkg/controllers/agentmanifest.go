package controllers

import (
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	workv1 "open-cluster-management.io/api/work/v1"
)

func newManifestWork(managedClusterAddon *addonv1alpha1.ManagedClusterAddOn, objects ...runtime.Object) *workv1.ManifestWork {
	owner := metav1.NewControllerRef(managedClusterAddon, addonv1alpha1.GroupVersion.WithKind("ManagedClusterAddOn"))

	work := &workv1.ManifestWork{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:       managedClusterAddon.Namespace,
			Name:            "addon-cluster-proxy-service-proxy",
			OwnerReferences: []metav1.OwnerReference{*owner},
		},
	}

	var manifests []workv1.Manifest
	for _, object := range objects {
		manifest := workv1.Manifest{}
		manifest.Object = object
		manifests = append(manifests, manifest)
	}
	work.Spec.Workload.Manifests = manifests

	return work
}

func newService(agentInstallNamespace string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: agentInstallNamespace,
			Name:      "cluster-proxy-service-proxy",
			Labels: map[string]string{
				"app": "cluster-proxy-service-proxy",
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "service-proxy",
					Port: 7443,
				},
			},
			Selector: map[string]string{
				"app": "cluster-proxy-service-proxy",
			},
		},
	}
}

func newServerCertSecret(agentInstallNamespace string, keyData, certData []byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: agentInstallNamespace,
			Name:      "cluster-proxy-service-proxy-server-certificates",
		},
		Data: map[string][]byte{
			"tls.crt": certData,
			"tls.key": keyData,
		},
	}
}

func newDeployment(agentInstallNamespace string,
	image string, imagePullPolicy corev1.PullPolicy,
	nodeSelector map[string]string,
	tolerations []corev1.Toleration,
) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: agentInstallNamespace,
			Name:      "cluster-proxy-service-proxy",
			Labels: map[string]string{
				"app": "cluster-proxy-service-proxy",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "cluster-proxy-service-proxy",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "cluster-proxy-service-proxy",
					},
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "apiserver-ca",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "kube-root-ca.crt",
									},
								},
							},
						},
						{
							Name: "ocpservice-ca",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "openshift-service-ca.crt",
									},
								},
							},
						},
						{
							Name: "service-proxy-server-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "cluster-proxy-service-proxy-server-certificates",
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "cluster-proxy-service-proxy",
							Image:           image,
							ImagePullPolicy: imagePullPolicy,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
							},
							Args: []string{
								"/cluster-proxy",
								"service-proxy",
								"--apiserver-ca=/apiserver-ca/ca.crt",
								"--ocpservice-ca=/ocpservice-ca/service-ca.crt",
								"--cert=/server-cert/tls.crt",
								"--key=/server-cert/tls.key",
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(8000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
								InitialDelaySeconds: 2,
								PeriodSeconds:       10,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "apiserver-ca",
									MountPath: "/apiserver-ca",
								},
								{
									Name:      "ocpservice-ca",
									MountPath: "/ocpservice-ca",
								},
								{
									Name:      "service-proxy-server-cert",
									MountPath: "/server-cert",
									ReadOnly:  true,
								},
							},
						},
					},
					ImagePullSecrets: []corev1.LocalObjectReference{
						{
							Name: imagePullSecret,
						},
					},
					Tolerations:  tolerations,
					NodeSelector: nodeSelector,
				},
			},
		},
	}
}
