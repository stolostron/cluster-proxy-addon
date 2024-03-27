package controllers

import (
	"context"
	"time"

	"github.com/openshift/library-go/pkg/crypto"
	"github.com/spf13/cobra"
	"github.com/stolostron/cluster-proxy-addon/pkg/constant"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1client "k8s.io/client-go/kubernetes/typed/core/v1"
	corev1listers "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/util/workqueue"
	certrotation "open-cluster-management.io/addon-framework/pkg/certrotation"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var _ reconcile.Reconciler = &reconcileServerCertificates{}

var (
	// certificatesNamespace is the namespace where signer secret and generated server certificates secret is stored.
	certificatesNamespace string
	signerSecretName      string
	signerSecretNamespace string
	agentImage            string
)

func addFlagsForCertController(cmd *cobra.Command) {
	cmd.Flags().StringVar(&certificatesNamespace, "certificates-namespace", "default", "The namespace where the secret is stored.")

	cmd.Flags().StringVar(&signerSecretName, "signer-secret-name", "cluster-proxy-signer", "The name of the secret that contains the signer certificate and key.") // the default value align with the signer-secret-name in manager-deployment.yaml.
	cmd.Flags().StringVar(&signerSecretNamespace, "signer-secret-namespace", "default", "The namespace where the secret is stored.")
	cmd.Flags().StringVar(&agentImage, "agent-image", "", "The image of agent") // TODO: remove this flag after the template in the backplane-operator repo is removed.
}

// reconcileServerCertificates sign certificates for the server with the signer ca created by the cluster-proxy.
type reconcileServerCertificates struct {
	client                                  client.Client
	serverCertRotation                      *certrotation.TargetRotation
	signerSecretName, signerSecretNamespace string
}

func registerCertController(certNamespace string,
	signerSecretName, signerSecretNamespace string,
	secertLister corev1listers.SecretLister,
	secertGetter corev1client.SecretsGetter, mgr manager.Manager) error {

	c, err := controller.New("cert-controller", mgr, controller.Options{
		Reconciler: &reconcileServerCertificates{
			client:                mgr.GetClient(),
			signerSecretName:      signerSecretName,
			signerSecretNamespace: signerSecretNamespace,
			serverCertRotation: &certrotation.TargetRotation{
				Namespace: certNamespace,
				Name:      constant.ServerCertSecretName,
				Validity:  time.Hour * 24 * 180, // align with the signer ca by cluster-proxy
				HostNames: []string{"*", "localhost", "127.0.0.1"},
				Lister:    secertLister,
				Client:    secertGetter,
			},
		},
	})
	if err != nil {
		return err
	}

	if err := c.Watch(source.Kind(mgr.GetCache(), &corev1.Secret{}), &secretHandler{
		signerSecretName:      signerSecretName,
		singerSecretNamespace: signerSecretNamespace,
	}); err != nil {
		return err
	}

	return nil
}

// Reconile reconcile the server certificates.
func (r *reconcileServerCertificates) Reconcile(context.Context, reconcile.Request) (reconcile.Result, error) {
	// get signer secret
	signerSecret := &corev1.Secret{}
	err := r.client.Get(context.TODO(), types.NamespacedName{
		Name:      r.signerSecretName,
		Namespace: r.signerSecretNamespace,
	}, signerSecret)
	if err != nil {
		return reconcile.Result{}, err
	}

	ca, err := crypto.GetCAFromBytes(signerSecret.Data["ca.crt"], signerSecret.Data["ca.key"])
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.serverCertRotation.EnsureTargetCertKeyPair(ca, ca.Config.Certs)
	if err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

type secretHandler struct {
	signerSecretName      string
	singerSecretNamespace string
}

var _ handler.EventHandler = &secretHandler{}

func (h *secretHandler) Create(ctx context.Context, evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	if !h.isSignerSecret(evt.Object) {
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Object.GetName(),
		Namespace: evt.Object.GetNamespace(),
	}})
}

func (h *secretHandler) Update(ctx context.Context, evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	if !h.isSignerSecret(evt.ObjectNew) {
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.ObjectNew.GetName(),
		Namespace: evt.ObjectNew.GetNamespace(),
	}})
}

func (h *secretHandler) Delete(ctx context.Context, evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	if !h.isSignerSecret(evt.Object) {
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Object.GetName(),
		Namespace: evt.Object.GetNamespace(),
	}})
}

func (h *secretHandler) Generic(ctx context.Context, evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	if !h.isSignerSecret(evt.Object) {
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Object.GetName(),
		Namespace: evt.Object.GetNamespace(),
	}})
}

func (h *secretHandler) isSignerSecret(o client.Object) bool {
	return o.GetName() == h.signerSecretName && o.GetNamespace() == h.singerSecretNamespace
}
