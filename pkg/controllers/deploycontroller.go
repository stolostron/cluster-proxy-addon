package controllers

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/stolostron/cluster-lifecycle-api/helpers/imageregistry"
	"github.com/stolostron/cluster-proxy-addon/pkg/constant"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	addonv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	addonv1alpha1lister "open-cluster-management.io/api/client/addon/listers/addon/v1alpha1"
	clusterv1lister "open-cluster-management.io/api/client/cluster/listers/cluster/v1"
	workv1client "open-cluster-management.io/api/client/work/clientset/versioned"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	workv1 "open-cluster-management.io/api/work/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func init() {
	utilruntime.Must(addonv1alpha1.AddToScheme(scheme))
	utilruntime.Must(workv1.AddToScheme(scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme))
}

const (
	imageName              = "cluster_proxy_addon"
	defaultImagePullPolicy = "IfNotPresent"
	imagePullSecret        = "open-cluster-management-image-pull-credentials"

	// annotationNodeSelector is key name of nodeSelector annotation synced from mch
	annotationNodeSelector = "open-cluster-management/nodeSelector"
)

var (
	agentImage string
)

func addFlagsForDeployController(cmd *cobra.Command) {
	cmd.Flags().StringVar(&agentImage, "agent-image", "", "The image of agent")
}

type reconcileDeployAgentManifestwork struct {
	client            client.Client
	addonLister       addonv1alpha1lister.ManagedClusterAddOnLister
	addonConfigLister addonv1alpha1lister.AddOnDeploymentConfigLister
	clusterLister     clusterv1lister.ManagedClusterLister
	workClient        workv1client.Interface
}

func registerDeployController(addonLister addonv1alpha1lister.ManagedClusterAddOnLister, addonConfigLister addonv1alpha1lister.AddOnDeploymentConfigLister, clusterLister clusterv1lister.ManagedClusterLister, workClient workv1client.Interface, mgr manager.Manager) error {
	c, err := controller.New("deploy-controller", mgr, controller.Options{
		Reconciler: &reconcileDeployAgentManifestwork{
			client:            mgr.GetClient(),
			addonLister:       addonLister,
			addonConfigLister: addonConfigLister,
			clusterLister:     clusterLister,
			workClient:        workClient,
		},
	})
	if err != nil {
		return err
	}

	if err := c.Watch(&source.Kind{Type: &addonv1alpha1.ManagedClusterAddOn{}}, &managedcluteraddonHandler{}); err != nil {
		return err
	}
	return nil
}

func (r *reconcileDeployAgentManifestwork) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	var err error

	// get managedClusterAddon name and managedClusterNamee
	managedClusterAddonName := request.Name
	managedClusterName := request.Namespace
	if len(managedClusterName) == 0 || len(managedClusterAddonName) == 0 {
		klog.Infof("Skip reconcile: managedClusterAddonName or managedClusterName is empty")
		return reconcile.Result{}, nil
	}

	// get managedCluster
	managedCluster, err := r.clusterLister.Get(managedClusterName)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("Skip reconcile: managedCluster %q not found", managedClusterName)
			return reconcile.Result{}, nil
		}
		klog.Errorf("Failed to get managedCluster %q: %v", managedClusterName, err)
		return reconcile.Result{}, err
	}
	if !managedCluster.DeletionTimestamp.IsZero() {
		klog.Infof("Skip reconcile: managedCluster %q is deleting", managedClusterName)
		return reconcile.Result{}, nil
	}

	// get managedClusterAddon
	managedClusterAddon, err := r.addonLister.ManagedClusterAddOns(managedClusterName).Get(managedClusterAddonName)
	if err != nil {
		if errors.IsNotFound(err) {
			klog.Infof("Skip reconcile: managedClusterAddon %q not found", managedClusterAddonName)
			return reconcile.Result{}, nil
		}
		klog.Errorf("Failed to get managedClusterAddon %q: %v", managedClusterAddonName, err)
		return reconcile.Result{}, err
	}
	if !managedClusterAddon.DeletionTimestamp.IsZero() {
		klog.Infof("Skip reconcile: managedClusterAddon %q is deleting", managedClusterAddonName)
		return reconcile.Result{}, nil
	}

	// get server certificates
	key, cert, err := getServerCertificatesFromSecret(r.client, certificatesNamespace)
	if err != nil {
		klog.Errorf("Failed to get server certificates: %v", err)
		return reconcile.Result{}, err
	}

	// get override image name
	overrideImageName, err := getOverrideImageName(managedCluster, agentImage)
	if err != nil {
		klog.Errorf("Failed to get override image name: %v", err)
		return reconcile.Result{}, err
	}

	var nodeSelector map[string]string
	tolerations := []corev1.Toleration{
		{
			Key:      "dedicated",
			Operator: corev1.TolerationOpEqual,
			Value:    "infra",
			Effect:   corev1.TaintEffectNoSchedule,
		},
		{
			Key:      "node-role.kubernetes.io/infra",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
	}

	// get nodeSelector
	nodeSelector, err = getLocalClusterNodeSelector(managedCluster)
	if err != nil {
		klog.Errorf("Failed to get nodeSelector: %v", err)
		return reconcile.Result{}, err
	}

	// get AddonDeploymentConfig
	for _, cr := range managedClusterAddon.Status.ConfigReferences {
		if cr.Resource == "addondeploymentconfigs" {
			// get nodeplacement
			addonDeploymentConfig, err := r.addonConfigLister.AddOnDeploymentConfigs(cr.Namespace).Get(cr.Name)
			if err != nil {
				if errors.IsNotFound(err) {
					klog.Infof("Skip reconcile: AddOnDeploymentConfig %q not found", cr.Name)
					break
				}
				klog.Errorf("Failed to get AddOnDeploymentConfig %q: %v", cr.Name, err)
				return reconcile.Result{}, err
			}

			if !addonDeploymentConfig.DeletionTimestamp.IsZero() {
				klog.Infof("Skip reconcile: AddOnDeploymentConfig %q is deleting", cr.Name)
				break
			}
			nodeplacement := addonDeploymentConfig.Spec.NodePlacement.DeepCopy()
			if nodeplacement != nil {
				// set nodeselector and tolerations based on nodeplacement
				nodeSelector = nodeplacement.NodeSelector
				tolerations = nodeplacement.Tolerations
			}
			break
		}
	}

	// new manifest service
	service := newService(constant.AgentInstallNamespace)

	// new manifest server certificates secret
	serverCertSecret := newServerCertSecret(constant.AgentInstallNamespace, key, cert)

	// new deployment
	deployment := newDeployment(constant.AgentInstallNamespace,
		overrideImageName, defaultImagePullPolicy,
		nodeSelector,
		tolerations,
	)

	// new agent manifestwork
	manifestWork := newManifestWork(managedClusterAddon, service, serverCertSecret, deployment)

	// create or update manifestwork
	if err := createOrUpdateManifestWork(ctx, r.workClient, manifestWork); err != nil {
		klog.Errorf("Failed to create or update manifestwork: %v", err)
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func createOrUpdateManifestWork(ctx context.Context, workClient workv1client.Interface, manifestWork *workv1.ManifestWork) error {
	// get manifestwork
	existingManifestWork, err := workClient.WorkV1().ManifestWorks(manifestWork.Namespace).Get(ctx, manifestWork.Name, metav1.GetOptions{})
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		// create manifestwork
		_, err := workClient.WorkV1().ManifestWorks(manifestWork.Namespace).Create(ctx, manifestWork, metav1.CreateOptions{})
		return err
	}

	// update manifestwork
	existingManifestWork.Spec = manifestWork.Spec
	_, err = workClient.WorkV1().ManifestWorks(manifestWork.Namespace).Update(ctx, existingManifestWork, metav1.UpdateOptions{})
	return err
}

func getServerCertificatesFromSecret(kubeClient client.Client, secretNamespace string) ([]byte, []byte, error) {
	secret := &corev1.Secret{}
	err := kubeClient.Get(context.TODO(), types.NamespacedName{
		Name:      constant.ServerCertSecretName,
		Namespace: secretNamespace,
	}, secret)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to get secret %s in the namespace %s: %v", constant.ServerCertSecretName, secretNamespace, err)

	}
	cert, ok := secret.Data["tls.crt"]
	if !ok {
		return nil, nil, fmt.Errorf("secret %s does not contain tls.crt", constant.ServerCertSecretName)
	}
	key, ok := secret.Data["tls.key"]
	if !ok {
		return nil, nil, fmt.Errorf("secret %s does not contain tls.key", constant.ServerCertSecretName)
	}
	return key, cert, nil
}

func getOverrideImageName(cluster *clusterv1.ManagedCluster, agentImage string) (string, error) {
	return imageregistry.OverrideImageByAnnotation(cluster.GetAnnotations(), agentImage)
}

func getLocalClusterNodeSelector(managedCluster *clusterv1.ManagedCluster) (map[string]string, error) {
	nodeSelector := map[string]string{}

	if managedCluster.GetName() == "local-cluster" {
		annotations := managedCluster.GetAnnotations()
		if nodeSelectorString, ok := annotations[annotationNodeSelector]; ok {
			if err := json.Unmarshal([]byte(nodeSelectorString), &nodeSelector); err != nil {
				klog.Error(err, "failed to unmarshal nodeSelector annotation of cluster %v", managedCluster.GetName())
				return nodeSelector, err
			}
		}
	}

	return nodeSelector, nil
}

type managedcluteraddonHandler struct{}

var _ handler.EventHandler = &managedcluteraddonHandler{}

func (h *managedcluteraddonHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if evt.Object.GetName() != constant.AddonName {
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Object.GetName(),
		Namespace: evt.Object.GetNamespace(),
	}})
}

func (h *managedcluteraddonHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if evt.ObjectNew.GetName() != constant.AddonName {
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.ObjectNew.GetName(),
		Namespace: evt.ObjectNew.GetNamespace(),
	}})
}

func (h *managedcluteraddonHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	// do nothing
}

func (h *managedcluteraddonHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if evt.Object.GetName() != constant.AddonName {
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Object.GetName(),
		Namespace: evt.Object.GetNamespace(),
	}})
}
