package controllers

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterclient "open-cluster-management.io/api/client/cluster/clientset/versioned"
	workclient "open-cluster-management.io/api/client/work/clientset/versioned"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

func init() {
	logger := klogr.New()
	log.SetLogger(logger)

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
}

var (
	scheme = runtime.NewScheme()

	metricsAddr, probeAddr string
	enableLeaderElection   bool
)

func NewControllersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "controllers",
		Short: "controllers",
		Run: func(cmd *cobra.Command, args []string) {
			err := runControllerManager()
			if err != nil {
				klog.Fatal(err, "unable to run controller manager")
			}
		},
	}

	addFlags(cmd)
	addFlagsForCertController(cmd)

	return cmd
}

func addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	cmd.Flags().StringVar(&probeAddr, "health-probe-addr", ":8081", "The address the probe endpoint binds to.")
	cmd.Flags().BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
}

func runControllerManager() error {
	ctx, cancel := context.WithCancel(signals.SetupSignalHandler())
	defer cancel()

	// Setup clients, informers and listers.
	kubeConfig := config.GetConfigOrDie()

	// In MCE 2.5, cluster-proxy-addon will delete the manifestwork that used to create the cluster-proxy-service-proxy
	// TODO: remove this function in 2.6. @xuezhaojun
	stopSignal := time.After(10 * time.Minute)
	go func() {
		for {
			select {
			case <-stopSignal:
				klog.Info("Stopping deletion of manifestwork")
				return
			default:
				err := deleteClusterProxyServiceProxy(ctx, kubeConfig)
				if err != nil {
					klog.Error(err, "error deleting manifestwork")
				}
				time.Sleep(1 * time.Minute)
			}
		}
	}()

	// secret client and lister
	nativeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	secertClient := nativeClient.CoreV1()
	informerFactory := informers.NewSharedInformerFactory(nativeClient, 10*time.Minute)
	secertLister := informerFactory.Core().V1().Secrets().Lister()

	go informerFactory.Start(ctx.Done())

	// New controller manager
	mgr, err := manager.New(kubeConfig, manager.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "cluster-proxy-service-proxy",
	})
	if err != nil {
		klog.Error(err, "unable to set up overall controller manager")
		return err
	}

	// Register CertController
	err = registerCertController(certificatesNamespace, signerSecretName, signerSecretNamespace, secertLister, secertClient, mgr)
	if err != nil {
		klog.Error(err, "unable to set up cert-controller")
		return err
	}

	if err := mgr.Start(ctx); err != nil {
		klog.Error(err, "problem running manager")
		return err
	}
	return nil
}

// In the previous version, cluster-proxy-addon create a manifestwork
func deleteClusterProxyServiceProxy(ctx context.Context, kubeconfig *rest.Config) error {
	klog.Info("delete manifestwork addon-cluster-proxy-service-proxy in managedclusters")
	clusterClient, err := clusterclient.NewForConfig(kubeconfig)
	if err != nil {
		return err
	}

	workClient, err := workclient.NewForConfig(kubeconfig)
	if err != nil {
		return err
	}

	managedclusters, err := clusterClient.ClusterV1().ManagedClusters().List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	klog.Infof("found %d managedclusters", len(managedclusters.Items))

	errs := []error{}
	for _, mc := range managedclusters.Items {
		// get manifestwork, if the manifestwork is not found, do nothing
		_, err := workClient.WorkV1().ManifestWorks(mc.Name).Get(ctx, "addon-cluster-proxy-service-proxy", metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				klog.Infof("manifestwork addon-cluster-proxy-service-proxy not found in managedcluster %s", mc.Name)
				continue
			}
			errs = append(errs, fmt.Errorf("failed to get manifestwork addon-cluster-proxy-service-proxy in managedcluster %s: %v", mc.Name, err))
			continue
		}

		// delete manifestwork
		err = workClient.WorkV1().ManifestWorks(mc.Name).Delete(ctx, "addon-cluster-proxy-service-proxy", metav1.DeleteOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to delete manifestwork addon-cluster-proxy-service-proxy in managedcluster %s: %v", mc.Name, err))
			continue
		}
	}

	return utilerrors.NewAggregate(errs)
}
