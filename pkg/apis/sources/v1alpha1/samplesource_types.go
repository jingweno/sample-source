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

package v1alpha1

import (
	"github.com/knative/pkg/apis/duck"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

var _ runtime.Object = (*SampleSource)(nil)

var _ = duck.VerifyType(&SampleSource{}, &duckv1alpha1.Conditions{})

const (
	SampleSourceConditionSinkProvided      duckv1alpha1.ConditionType = "SinkProvided"
	SampleSourceConditionServiceDeployed   duckv1alpha1.ConditionType = "ServiceDeployed"
	SampleSourceConditionServiceRouteReady duckv1alpha1.ConditionType = "RouteDeployed"
)

var sampleSourceCondSet = duckv1alpha1.NewLivingConditionSet(
	SampleSourceConditionSinkProvided,
	SampleSourceConditionServiceDeployed,
	SampleSourceConditionServiceRouteReady,
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SampleSourceSpec defines the desired state of SampleSource
type SampleSourceSpec struct {
	// Sink is a reference to an object that will resolve to a domain name to use
	// as the sink.
	// +optional
	Sink *corev1.ObjectReference `json:"sink,omitempty"`
}

// SampleSourceStatus defines the observed state of SampleSource
type SampleSourceStatus struct {
	// inherits duck/v1alpha1 Status, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last processed by the controller.
	// * Conditions - the latest available observations of a resource's current state.
	duckv1alpha1.Status `json:",inline"`

	// SinkURI is the current active sink URI that has been configured for the SampleSource.
	// +optional
	SinkURI string `json:"sinkURI,omitempty"`

	// Route is the route to the function
	// +optional
	Route string `json:"route,omitempty"`
}

func (s *SampleSourceStatus) InitializeConditions() {
	sampleSourceCondSet.Manage(s).InitializeConditions()
}

func (s *SampleSourceStatus) MarkSink(uri string) {
	s.SinkURI = uri
	if len(uri) > 0 {
		sampleSourceCondSet.Manage(s).MarkTrue(SampleSourceConditionSinkProvided)
	} else {
		sampleSourceCondSet.Manage(s).MarkUnknown(SampleSourceConditionSinkProvided,
			"SinkEmpty", "Sink has resolved to empty.")
	}
}

func (s *SampleSourceStatus) MarkNoSink(reason, messageFormat string, messageA ...interface{}) {
	sampleSourceCondSet.Manage(s).MarkFalse(SampleSourceConditionSinkProvided, reason, messageFormat, messageA...)
}

func (s *SampleSourceStatus) MarkServiceDeployed() {
	sampleSourceCondSet.Manage(s).MarkTrue(SampleSourceConditionServiceDeployed)
}

func (s *SampleSourceStatus) IsServiceDeployed() bool {
	cond := sampleSourceCondSet.Manage(s).GetCondition(SampleSourceConditionServiceDeployed)
	if cond != nil && cond.Status == corev1.ConditionTrue {
		return true
	}

	return false
}

func (s *SampleSourceStatus) MarkRoute(route string) {
	s.Route = route
	if len(route) > 0 {
		sampleSourceCondSet.Manage(s).MarkTrue(SampleSourceConditionServiceRouteReady)
	} else {
		sampleSourceCondSet.Manage(s).MarkUnknown(SampleSourceConditionServiceRouteReady,
			"RouteEmpty", "Route has resolved to empty.")
	}
}

func (s *SampleSourceStatus) MarkNoRoute(reason, messageFormat string, messageA ...interface{}) {
	sampleSourceCondSet.Manage(s).MarkFalse(SampleSourceConditionServiceRouteReady, reason, messageFormat, messageA...)
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SampleSource is the Schema for the samplesources API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type SampleSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SampleSourceSpec   `json:"spec,omitempty"`
	Status SampleSourceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SampleSourceList contains a list of SampleSource
type SampleSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SampleSource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SampleSource{}, &SampleSourceList{})
}
