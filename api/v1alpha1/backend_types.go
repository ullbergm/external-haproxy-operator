/*
Copyright 2025.

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
	"fmt"

	haproxy_models "github.com/haproxytech/client-native/v6/models"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// BackendSpec defines the desired state of Backend.
type BackendSpec struct {
	// name
	// Required: true
	// Pattern: ^[A-Za-z0-9-_.:]+$
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9-_.:]+$`
	Name string `json:"name"`

	// balance
	Balance *Balance `json:"balance,omitempty"`

	// adv check
	// Enum: ["httpchk","ldap-check","mysql-check","pgsql-check","redis-check","smtpchk","ssl-hello-chk","tcp-check"]
	// +kubebuilder:validation:Enum=httpchk;ldap-check;mysql-check;pgsql-check;redis-check;smtpchk;ssl-hello-chk;tcp-check;
	AdvCheck string `json:"adv_check,omitempty"`

	// servers
	Servers Servers `json:"servers,omitempty"`

	// HTTP check list
	HTTPCheckList HTTPChecks `json:"http_check_list,omitempty"`
}

type Balance struct {
	// algorithm
	// Required: true
	// Enum: ["first","hash","hdr","leastconn","random","rdp-cookie","roundrobin","source","static-rr","uri","url_param"]
	// +kubebuilder:validation:Enum=first;hash;hdr;leastconn;random;rdp-cookie;roundrobin;source;static-rr;uri;url_param;
	Algorithm *string `json:"algorithm"`
}

// HAProxy backend servers array.
type Servers []*Server

// HAProxy backend server configuration
// Example: {"address":"10.1.1.1","name":"www","port":8080}
type Server struct {
	ServerParams `json:",inline"`

	// address
	// Optional: If ValueFrom is not set, Address is required.
	// Pattern: ^[^\s]+$
	// +kubebuilder:validation:Pattern=`^[^\s]+$`
	Address string `json:"address,omitempty"`

	// id
	ID *int64 `json:"id,omitempty"`

	// name
	// Optional: If ValueFrom is not set, Name is required.
	// Pattern: ^[^\s]+$
	// +kubebuilder:validation:Pattern=`^[^\s]+$`
	Name string `json:"name,omitempty"`

	// port
	// Maximum: 65535
	// Minimum: 1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:Minimum=1
	Port *int64 `json:"port,omitempty"`

	// valueFrom
	// Specify a source to populate the server's address/port from another resource (e.g., a Kubernetes Service).
	// Optional: If Address/Name are not set, ValueFrom must be set.
	ValueFrom *ServerValueFromSource `json:"valueFrom,omitempty"`
	// Note: Either Address/Name or ValueFrom must be set. This is enforced at runtime, not by OpenAPI validation.
}

// Validate checks that either Address/Name or ValueFrom is set, but not both.
func (s *Server) Validate() error {
	if (s.Address == "" || s.Name == "") && s.ValueFrom == nil {
		return fmt.Errorf("either address/name or valueFrom must be set for server")
	}
	if (s.Address != "" || s.Name != "") && s.ValueFrom != nil {
		return fmt.Errorf("only one of address/name or valueFrom may be set for server")
	}
	return nil
}

// ValidateServers checks all servers in a Servers slice.
func ValidateServers(servers Servers) error {
	for i, s := range servers {
		if err := s.Validate(); err != nil {
			return fmt.Errorf("server[%d]: %w", i, err)
		}
	}
	return nil
}

// ServerValueFromSource represents a source for the server's value (address/port) from another resource.
type ServerValueFromSource struct {
	// serviceRef
	// Reference to a Kubernetes Service to dynamically resolve endpoints.
	ServiceRef *K8sServiceRef `json:"serviceRef,omitempty"`
}

// K8sServiceRef allows referencing a Kubernetes Service object for dynamic endpoint resolution.
type K8sServiceRef struct {
	// namespace
	// Namespace of the Service. Defaults to the backend resource's namespace if omitted.
	Namespace string `json:"namespace,omitempty"`

	// name
	// Required: true
	// Name of the Service to reference.
	// +kubebuilder:validation:Pattern=`^[A-Za-z0-9-_.:]+$`
	Name string `json:"name"`

	// port
	// Name or number of the port to use from the Service.
	Port string `json:"port,omitempty"`
}

type ServerParams struct {
	// check
	// Enum: ["enabled","disabled"]
	// +kubebuilder:validation:Enum=enabled;disabled;
	Check string `json:"check,omitempty"`
}

// BackendStatus defines the observed state of Backend.
type BackendStatus struct {
	// Represents the observations of a backend's current state.
	// Backend.status.conditions.type are: "Available", "Progressing", and "Degraded"
	// Backend.status.conditions.status are one of True, False, Unknown.
	// Backend.status.conditions.reason the value should be a CamelCase string and producers of specific
	// condition types may define expected values and meanings for this field, and whether the values
	// are considered a guaranteed API.
	// Backend.status.conditions.Message is a human readable message indicating details about the transition.

	// Conditions store the status conditions of the Backend
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`
}

// HTTPChecks is a slice of HTTPCheck pointers.
type HTTPChecks []*HTTPCheck

// HTTPCheck defines the HTTP check configuration for a backend.
type HTTPCheck struct {
	// type
	// Required: true
	// Enum: ["comment","connect","disable-on-404","expect","send","send-state","set-var","set-var-fmt","unset-var"]
	// +kubebuilder:validation:Enum=comment;connect;disable-on-404;expect;send;send-state;set-var;set-var-fmt;unset-var;
	Type string `json:"type"`

	// method
	// Enum: ["HEAD","PUT","POST","GET","TRACE","PATCH","DELETE","CONNECT","OPTIONS"]
	// +kubebuilder:validation:Enum=HEAD;PUT;POST;GET;TRACE;PATCH;DELETE;CONNECT;OPTIONS;
	Method string `json:"method,omitempty"`

	// uri
	URI string `json:"uri,omitempty"`

	// check headers
	CheckHeaders []*ReturnHeader `json:"headers,omitempty"`
}

// ReturnHeader defines the headers sent/returned by the HTTP check.
type ReturnHeader struct {

	// fmt
	// Required: true
	Fmt *string `json:"fmt"`

	// name
	// Required: true
	Name *string `json:"name"`
}

// ----- Conversion helpers -----

func BackendSpecToModel(spec BackendSpec) *haproxy_models.Backend {
	var balance *haproxy_models.Balance
	if spec.Balance != nil {
		balance = &haproxy_models.Balance{
			Algorithm: spec.Balance.Algorithm,
		}
	}

	servers := make(map[string]haproxy_models.Server)
	for _, s := range spec.Servers {
		servers[s.Name] = haproxy_models.Server{
			Name:    s.Name,
			Address: s.Address,
			Port:    s.Port,
			ID:      s.ID,
		}
	}

	httpChecks := make([]*haproxy_models.HTTPCheck, 0, len(spec.HTTPCheckList))
	for _, hc := range spec.HTTPCheckList {
		var headers []*haproxy_models.ReturnHeader
		for _, h := range hc.CheckHeaders {
			headers = append(headers, &haproxy_models.ReturnHeader{
				Name: h.Name,
				Fmt:  h.Fmt,
			})
		}
		httpChecks = append(httpChecks, &haproxy_models.HTTPCheck{
			Type:         hc.Type,
			Method:       hc.Method,
			URI:          hc.URI,
			CheckHeaders: headers,
		})
	}

	return &haproxy_models.Backend{
		BackendBase: haproxy_models.BackendBase{
			Name:     spec.Name,
			Balance:  balance,
			AdvCheck: spec.AdvCheck,
		},
		Servers:       servers,
		HTTPCheckList: httpChecks,
	}
}

func ModelToBackendSpec(model *haproxy_models.Backend) BackendSpec {
	var balance *Balance
	if model.Balance != nil {
		balance = &Balance{
			Algorithm: model.Balance.Algorithm,
		}
	}

	servers := make(Servers, 0, len(model.Servers))
	for _, s := range model.Servers {
		server := &Server{
			Name:    s.Name,
			Address: s.Address,
			Port:    s.Port,
			ID:      s.ID,
		}
		servers = append(servers, server)
	}

	httpChecks := make(HTTPChecks, 0, len(model.HTTPCheckList))
	for _, hc := range model.HTTPCheckList {
		var headers []*ReturnHeader
		for _, h := range hc.CheckHeaders {
			headers = append(headers, &ReturnHeader{
				Name: h.Name,
				Fmt:  h.Fmt,
			})
		}
		httpChecks = append(httpChecks, &HTTPCheck{
			Type:         hc.Type,
			Method:       hc.Method,
			URI:          hc.URI,
			CheckHeaders: headers,
		})
	}

	return BackendSpec{
		Name:          model.Name,
		Balance:       balance,
		AdvCheck:      model.AdvCheck,
		Servers:       servers,
		HTTPCheckList: httpChecks,
	}
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Backend is the Schema for the backends API.
// +kubebuilder:subresource:status
type Backend struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BackendSpec   `json:"spec,omitempty"`
	Status BackendStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BackendList contains a list of Backend.
type BackendList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Backend `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Backend{}, &BackendList{})
}
