package operator

import (
	"context"
	"k8s.io/client-go/kubernetes"

	operatorconfigclient "github.com/openshift/cluster-kube-descheduler-operator/pkg/generated/clientset/versioned"
	deschedulerv1beta1 "github.com/openshift/cluster-kube-descheduler-operator/pkg/apis/descheduler/v1beta1"
	"github.com/openshift/cluster-kube-descheduler-operator/pkg/operator/configobservation"
	"github.com/openshift/cluster-kube-descheduler-operator/pkg/operator/resourcesynccontroller"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/openshift/library-go/pkg/operator/genericoperatorclient"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
)

const (
	workQueueKey = "key"
)

func RunOperator(ctx context.Context, cc *controllercmd.ControllerContext) error {
	kubeClient, err := kubernetes.NewForConfig(cc.ProtoKubeConfig)
	if err != nil {
		return err
	}

	kubeInformersForNamespaces := v1helpers.NewKubeInformersForNamespaces(kubeClient,
		"",
		"openshift-kube-descheduler-operator",
	)

	operatorConfigClient, err  := operatorconfigclient.NewForConfig(cc.KubeConfig)
	if err != nil {
		return err
	}

	operatorClient, dynamicInformers, err := genericoperatorclient.NewClusterScopedOperatorClient(cc.KubeConfig, deschedulerv1beta1.SchemeGroupVersion.WithResource("kubedeschedulers"))
	if err != nil {
		return err
	}

	resourceSyncController, err := resourcesynccontroller.NewResourceSyncController(
		operatorClient,
		kubeInformersForNamespaces,
		kubeClient,
		cc.EventRecorder,
	)
	if err != nil {
		return err
	}

	configObserver := configobservation.NewConfigObserver(
		operatorClient,
		kubeInformersForNamespaces,
		resourceSyncController,
		cc.EventRecorder,
	)

	targetConfigReconciler := NewTargetConfigReconciler(
		operatorConfigClient.KubedeschedulersV1beta1(),
		kubeClient,
		cc.EventRecorder,
	)

	kubeInformersForNamespaces.Start(ctx.Done())
	dynamicInformers.Start(ctx.Done())

	go resourceSyncController.Run(ctx, 1)
	go targetConfigReconciler.Run(1, ctx.Done())
	go configObserver.Run(ctx, 1)

	return nil
}
