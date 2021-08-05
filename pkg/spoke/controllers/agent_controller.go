package controllers

import (
	"context"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	corev1informers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	corev1lister "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	"open-cluster-management.io/cluster-proxy-addon/pkg/config"
)

type agentController struct {
	spokeKubeClient    kubernetes.Interface
	hunConfigMapLister corev1lister.ConfigMapLister
	clusterName        string
	recorder           events.Recorder
}

func NewAgentController(
	spokeKubeClient kubernetes.Interface,
	configmapInformers corev1informers.ConfigMapInformer,
	recorder events.Recorder) factory.Controller {
	c := &agentController{
		spokeKubeClient:    spokeKubeClient,
		hunConfigMapLister: configmapInformers.Lister(),
		recorder:           recorder,
	}
	return factory.New().
		WithInformersQueueKeyFunc(func(object runtime.Object) string {
			key, _ := cache.MetaNamespaceKeyFunc(object)
			return key
		}, configmapInformers.Informer()).
		WithSync(c.sync).
		ToController(config.ADDON_AGENT_NAME, recorder)
}

func (c *agentController) sync(ctx context.Context, syncCtx factory.SyncContext) error {
	key := syncCtx.QueueKey()
	klog.V(4).Infof("Reconciling addon deploy %q", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		// ignore addon whose key is not in format: namespace/name
		return nil
	}

	if namespace != "open-cluster-management" || name != config.CaBundleConfigmap {
		return nil
	}

	cm, err := c.hunConfigMapLister.ConfigMaps(namespace).Get(name)
	switch {
	case errors.IsNotFound(err):
		return nil
	case err != nil:
		return err
	}

	configmap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cm.Name,
			Namespace: config.DEFAULT_NAMESPACE,
		},
		Data: cm.Data,
	}

	_, _, err = resourceapply.ApplyConfigMap(ctx, c.spokeKubeClient.CoreV1(), c.recorder, configmap)

	// TODO restart all deployments
	return err
}
