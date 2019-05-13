/*
Copyright 2019 The Knative Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package samplesource

import (
	"context"
	"fmt"

	sourcesv1alpha1 "github.com/heroku/sample-source/pkg/apis/sources/v1alpha1"
	"github.com/knative/eventing-sources/pkg/controller/sdk"
	"github.com/knative/eventing-sources/pkg/controller/sinks"
	"github.com/knative/pkg/logging"
	servingv1alpha1 "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller")

const (
	controllerAgentName = "http-source-controller"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new SampleSource Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	p := &sdk.Provider{
		AgentName: controllerAgentName,
		Parent:    &sourcesv1alpha1.SampleSource{},
		Owns:      []runtime.Object{&servingv1alpha1.Service{}},
		Reconciler: &ReconcileSampleSource{
			recorder:            mgr.GetRecorder(controllerAgentName),
			scheme:              mgr.GetScheme(),
			receiveAdapterImage: "jingweno/httpadapter-6b07a24d22ba8ab911a859aba268b488",
		},
	}
	return p.Add(mgr)
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("samplesource-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to SampleSource
	err = c.Watch(&source.Kind{Type: &sourcesv1alpha1.SampleSource{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// ReconcileSampleSource reconciles a SampleSource object
type ReconcileSampleSource struct {
	client              client.Client
	recorder            record.EventRecorder
	scheme              *runtime.Scheme
	receiveAdapterImage string
}

// Reconcile reads that state of the cluster for a SampleSource object and makes changes based on the state read
// and what is in the SampleSource.Spec
// +kubebuilder:rbac:groups=sources.knative.dev,resources=samplesources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sources.knative.dev,resources=samplesources/status,verbs=get;update;patch
func (r *ReconcileSampleSource) Reconcile(ctx context.Context, object runtime.Object) error {
	logger := logging.FromContext(ctx)
	source, ok := object.(*sourcesv1alpha1.SampleSource)
	if !ok {
		logger.Errorf("could not find http source %v\n", object)
		return nil
	}

	// See if the source has been deleted
	accessor, err := meta.Accessor(source)
	if err != nil {
		logger.Warnf("Failed to get metadata accessor: %s", zap.Error(err))
		return err
	}

	if accessor.GetDeletionTimestamp() == nil {
		return r.reconcile(ctx, source, logger)
	}

	return nil
}

func (r *ReconcileSampleSource) reconcile(ctx context.Context, source *sourcesv1alpha1.SampleSource, logger *zap.SugaredLogger) error {
	source.Status.InitializeConditions()

	sinkURI, err := sinks.GetSinkURI(ctx, r.client, source.Spec.Sink, source.Namespace)
	if err != nil {
		source.Status.MarkNoSink("Not found", "%s", err)
		return fmt.Errorf("Failed to get sink URI: %v", err)
	}
	logger.Infof("Updating source status sink uri to %s", sinkURI)
	source.Status.MarkSink(sinkURI)

	ksvc, err := r.getOwnedService(ctx, source, logger)
	if err != nil {
		if apierrors.IsNotFound(err) && !source.Status.IsServiceDeployed() {
			logger.Info("Making ksvc")
			ksvc = makeService(source, r.receiveAdapterImage)
			if err := controllerutil.SetControllerReference(source, ksvc, r.scheme); err != nil {
				return err
			}
			logger.Info("Creating ksvc")
			if err := r.client.Create(ctx, ksvc); err != nil {
				return err
			}

			source.Status.MarkServiceDeployed()
		}

		return err
	}

	routeCondition := ksvc.Status.GetCondition(servingv1alpha1.ServiceConditionRoutesReady)
	receiveAdapterDomain := ksvc.Status.Domain
	if routeCondition != nil && routeCondition.Status == corev1.ConditionTrue && receiveAdapterDomain != "" {
		logger.Infof("Updating source status route to %s", receiveAdapterDomain)
		source.Status.MarkRoute(receiveAdapterDomain)
	} else {
		source.Status.MarkNoRoute("No found", "")
	}

	return nil
}

func makeService(source *sourcesv1alpha1.SampleSource, receiveAdapterImage string) *servingv1alpha1.Service {
	labels := map[string]string{
		"receive-adapter": "http",
	}
	sinkURI := source.Status.SinkURI
	env := []corev1.EnvVar{
		{
			Name:  "PORT",
			Value: "5000",
		},
		{
			Name:  "SINK_URI",
			Value: sinkURI,
		},
	}

	return &servingv1alpha1.Service{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", source.Name),
			Namespace:    source.Namespace,
			Labels:       labels,
		},
		Spec: servingv1alpha1.ServiceSpec{
			RunLatest: &servingv1alpha1.RunLatestType{
				Configuration: servingv1alpha1.ConfigurationSpec{
					RevisionTemplate: servingv1alpha1.RevisionTemplateSpec{
						Spec: servingv1alpha1.RevisionSpec{
							Container: corev1.Container{
								Image: receiveAdapterImage,
								Env:   env,
							},
						},
					},
				},
			},
		},
	}
}

func (r *ReconcileSampleSource) getOwnedService(ctx context.Context, source *sourcesv1alpha1.SampleSource, logger *zap.SugaredLogger) (*servingv1alpha1.Service, error) {
	list := &servingv1alpha1.ServiceList{}
	err := r.client.List(ctx, &client.ListOptions{
		Namespace:     source.Namespace,
		LabelSelector: labels.Everything(),
		// TODO this is here because the fake client needs it.
		// Remove this when it's no longer needed.
		Raw: &metav1.ListOptions{
			TypeMeta: metav1.TypeMeta{
				APIVersion: servingv1alpha1.SchemeGroupVersion.String(),
				Kind:       "Service",
			},
		},
	},
		list)
	if err != nil {
		return nil, err
	}
	for _, ksvc := range list.Items {
		logger.Infof("Found ksvc %s", ksvc.Name)
		if metav1.IsControlledBy(&ksvc, source) {
			//TODO if there are >1 controlled, delete all but first?
			return &ksvc, nil
		}
	}
	return nil, apierrors.NewNotFound(servingv1alpha1.Resource("services"), "")
}

func (r *ReconcileSampleSource) InjectClient(c client.Client) error {
	r.client = c
	return nil
}
