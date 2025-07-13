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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	haproxy_models "github.com/haproxytech/client-native/v6/models"
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
	Servers map[string]Server `json:"servers,omitempty"`

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

// # HAProxy backend servers array
//
// swagger:model servers
type Servers []*Server

// HAProxy backend server configuration
// Example: {"address":"10.1.1.1","name":"www","port":8080}
//
// swagger:model server
type Server struct {
	ServerParams `json:",inline"`

	// address
	// Required: true
	// Pattern: ^[^\s]+$
	// +kubebuilder:validation:Pattern=`^[^\s]+$`
	Address string `json:"address"`

	// id
	ID *int64 `json:"id,omitempty"`

	// name
	// Required: true
	// Pattern: ^[^\s]+$
	// +kubebuilder:validation:Pattern=`^[^\s]+$`
	Name string `json:"name"`

	// port
	// Maximum: 65535
	// Minimum: 1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:Minimum=1
	Port *int64 `json:"port,omitempty"`
}

// swagger:model server_params
type ServerParams struct {
	// check
	// Enum: ["enabled","disabled"]
	// +kubebuilder:validation:Enum=enabled;disabled;
	Check string `json:"check,omitempty"`
}

// BackendStatus defines the observed state of Backend.
type BackendStatus struct {
}

// swagger:model http_checks
type HTTPChecks []*HTTPCheck

// swagger:model http_check
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

// swagger:model ReturnHeader
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
	for name, s := range spec.Servers {
		servers[name] = haproxy_models.Server{
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

	servers := make(map[string]Server)
	for name, s := range model.Servers {
		servers[name] = Server{
			Name:    s.Name,
			Address: s.Address,
			Port:    s.Port,
			ID:      s.ID,
		}
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
