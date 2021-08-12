package controllers

import (
	"context"
	"time"

	"open-cluster-management.io/registration-operator/pkg/certrotation"

	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
	errorhelpers "github.com/openshift/library-go/pkg/operator/v1helpers"

	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	signerSecret           = "cluster-proxy-signer"
	caBundleConfigmap      = "cluster-proxy-ca-bundle"
	clusterProxyAddOnSecet = "cluster-proxy-addon-serving-cert"
	signerNamePrefix       = "cluster-proxy-addon"
)

// Follow the rules below to set the value of SigningCertValidity/TargetCertValidity/ResyncInterval:
//
// 1) SigningCertValidity * 1/5 * 1/5 > ResyncInterval * 2
// 2) TargetCertValidity * 1/5 > ResyncInterval * 2
var SigningCertValidity = time.Hour * 24 * 365
var TargetCertValidity = time.Hour * 24 * 30
var ResyncInterval = time.Minute * 5

type certRotationController struct {
	signingRotation  certrotation.SigningRotation
	caBundleRotation certrotation.CABundleRotation
	targetRotations  []certrotation.TargetRotation
}

func NewCertRotationController(
	namespace string,
	anpRouteHost string,
	kubeClient kubernetes.Interface,
	secretInformer corev1informers.SecretInformer,
	configMapInformer corev1informers.ConfigMapInformer,
	recorder events.Recorder) factory.Controller {

	c := &certRotationController{
		signingRotation: certrotation.SigningRotation{
			Namespace:        namespace,
			Name:             signerSecret,
			SignerNamePrefix: signerNamePrefix,
			Validity:         SigningCertValidity,
			Lister:           secretInformer.Lister(),
			Client:           kubeClient.CoreV1(),
			EventRecorder:    recorder,
		},
		caBundleRotation: certrotation.CABundleRotation{
			Namespace:     namespace,
			Name:          caBundleConfigmap,
			Lister:        configMapInformer.Lister(),
			Client:        kubeClient.CoreV1(),
			EventRecorder: recorder,
		},
		targetRotations: []certrotation.TargetRotation{
			{
				Namespace:     namespace,
				Name:          clusterProxyAddOnSecet,
				Validity:      TargetCertValidity,
				HostNames:     []string{anpRouteHost},
				Lister:        secretInformer.Lister(),
				Client:        kubeClient.CoreV1(),
				EventRecorder: recorder,
			},
		},
	}

	return factory.New().
		ResyncEvery(ResyncInterval).
		WithSync(c.sync).
		WithInformers(secretInformer.Informer(), configMapInformer.Informer()).
		ToController("CertRotationController", recorder)
}

func (c certRotationController) sync(ctx context.Context, syncCtx factory.SyncContext) error {
	// reconcile cert/key pair for signer
	signingCertKeyPair, err := c.signingRotation.EnsureSigningCertKeyPair()
	if err != nil {
		return err
	}

	// reconcile ca bundle
	cabundleCerts, err := c.caBundleRotation.EnsureConfigMapCABundle(signingCertKeyPair)
	if err != nil {
		return err
	}

	// reconcile target cert/key pairs
	errs := []error{}
	for _, targetRotation := range c.targetRotations {
		if err := targetRotation.EnsureTargetCertKeyPair(signingCertKeyPair, cabundleCerts); err != nil {
			errs = append(errs, err)
		}
	}
	return errorhelpers.NewMultiLineAggregate(errs)
}
