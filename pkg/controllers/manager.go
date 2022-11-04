package controllers

import (
	"context"
	"time"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	addonv1alpha1client "open-cluster-management.io/api/client/addon/clientset/versioned"
	addoninformers "open-cluster-management.io/api/client/addon/informers/externalversions"
	clusterv1client "open-cluster-management.io/api/client/cluster/clientset/versioned"
	clusterv1informers "open-cluster-management.io/api/client/cluster/informers/externalversions"
	workv1client "open-cluster-management.io/api/client/work/clientset/versioned"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
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
	addFlagsForDeployController(cmd)

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

	// secret client and lister
	nativeClient, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	secertClient := nativeClient.CoreV1()
	informerFactory := informers.NewSharedInformerFactory(nativeClient, 10*time.Minute)
	secertLister := informerFactory.Core().V1().Secrets().Lister()

	go informerFactory.Start(ctx.Done())

	// addonClient and lister
	addonClient, err := addonv1alpha1client.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	addonInformerFactory := addoninformers.NewSharedInformerFactory(addonClient, 10*time.Minute)
	addonLister := addonInformerFactory.Addon().V1alpha1().ManagedClusterAddOns().Lister()

	go addonInformerFactory.Start(ctx.Done())

	// clusterClient and lister
	clusterClient, err := clusterv1client.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}
	clusterInformerFactory := clusterv1informers.NewSharedInformerFactory(clusterClient, 10*time.Minute)
	clusterLister := clusterInformerFactory.Cluster().V1().ManagedClusters().Lister()

	go clusterInformerFactory.Start(ctx.Done())

	// workClient
	workClient, err := workv1client.NewForConfig(kubeConfig)
	if err != nil {
		return err
	}

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

	// Register DeployController
	err = registerDeployController(addonLister, clusterLister, workClient, mgr)
	if err != nil {
		klog.Error(err, "unable to set up deploy-controller")
		return err
	}

	if err := mgr.Start(ctx); err != nil {
		klog.Error(err, "problem running manager")
		return err
	}
	return nil
}
