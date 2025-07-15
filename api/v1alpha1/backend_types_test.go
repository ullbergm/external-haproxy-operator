package v1alpha1

import (
	"reflect"
	"testing"

	haproxy_models "github.com/haproxytech/client-native/v6/models"
)

func TestBackendSpecToModel_BasicFields(t *testing.T) {
	alg := "roundrobin"
	spec := BackendSpec{
		Name:     "backend1",
		Balance:  &Balance{Algorithm: &alg},
		AdvCheck: "httpchk",
	}

	model := BackendSpecToModel(spec)

	if model.Name != spec.Name {
		t.Errorf("expected Name %q, got %q", spec.Name, model.Name)
	}
	if model.Balance == nil || model.Balance.Algorithm == nil || *model.Balance.Algorithm != alg {
		t.Errorf("expected Balance Algorithm %q, got %v", alg, model.Balance)
	}
	if model.AdvCheck != spec.AdvCheck {
		t.Errorf("expected AdvCheck %q, got %q", spec.AdvCheck, model.AdvCheck)
	}
}

func TestBackendSpecToModel_Servers(t *testing.T) {
	port := int64(8080)
	id := int64(1)
	spec := BackendSpec{
		Name: "backend2",
		Servers: Servers{
			&Server{
				Name:    "srv1",
				Address: "10.0.0.1",
				Port:    &port,
				ID:      &id,
			},
			&Server{
				Name:    "srv2",
				Address: "10.0.0.2",
			},
		},
	}

	model := BackendSpecToModel(spec)

	if len(model.Servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(model.Servers))
	}
	srv1, ok := model.Servers["srv1"]
	if !ok {
		t.Fatalf("srv1 not found in model.Servers")
	}
	if srv1.Address != "10.0.0.1" || srv1.Port == nil || *srv1.Port != port || srv1.ID == nil || *srv1.ID != id {
		t.Errorf("srv1 fields mismatch: %+v", srv1)
	}
	srv2, ok := model.Servers["srv2"]
	if !ok {
		t.Fatalf("srv2 not found in model.Servers")
	}
	if srv2.Address != "10.0.0.2" {
		t.Errorf("srv2 Address mismatch: got %q", srv2.Address)
	}
}

func TestBackendSpecToModel_HTTPCheckList(t *testing.T) {
	fmtVal := "fmtval"
	nameVal := "nameval"
	spec := BackendSpec{
		Name: "backend3",
		HTTPCheckList: HTTPChecks{
			&HTTPCheck{
				Type:   "expect",
				Method: "GET",
				URI:    "/health",
				CheckHeaders: []*ReturnHeader{
					{Name: &nameVal, Fmt: &fmtVal},
				},
			},
		},
	}

	model := BackendSpecToModel(spec)

	if len(model.HTTPCheckList) != 1 {
		t.Fatalf("expected 1 HTTPCheck, got %d", len(model.HTTPCheckList))
	}
	hc := model.HTTPCheckList[0]
	if hc.Type != "expect" || hc.Method != "GET" || hc.URI != "/health" {
		t.Errorf("HTTPCheck fields mismatch: %+v", hc)
	}
	if len(hc.CheckHeaders) != 1 {
		t.Fatalf("expected 1 header, got %d", len(hc.CheckHeaders))
	}
	header := hc.CheckHeaders[0]
	if header.Name == nil || *header.Name != nameVal || header.Fmt == nil || *header.Fmt != fmtVal {
		t.Errorf("Header fields mismatch: %+v", header)
	}
}

func TestBackendSpecToModel_EmptyFields(t *testing.T) {
	spec := BackendSpec{
		Name: "backend4",
	}
	model := BackendSpecToModel(spec)

	if model.Name != "backend4" {
		t.Errorf("expected Name backend4, got %q", model.Name)
	}
	if model.Balance != nil {
		t.Errorf("expected nil Balance, got %+v", model.Balance)
	}
	if model.AdvCheck != "" {
		t.Errorf("expected empty AdvCheck, got %q", model.AdvCheck)
	}
	if len(model.Servers) != 0 {
		t.Errorf("expected 0 servers, got %d", len(model.Servers))
	}
	if len(model.HTTPCheckList) != 0 {
		t.Errorf("expected 0 HTTPCheckList, got %d", len(model.HTTPCheckList))
	}
}

func TestBackendSpecToModel_FullRoundTrip(t *testing.T) {
	alg := "leastconn"
	port := int64(443)
	id := int64(42)
	fmtVal := "fmt"
	nameVal := "headername"
	spec := BackendSpec{
		Name:     "backend5",
		Balance:  &Balance{Algorithm: &alg},
		AdvCheck: "tcp-check",
		Servers: Servers{
			&Server{
				Name:    "srvA",
				Address: "192.168.1.1",
				Port:    &port,
				ID:      &id,
			},
		},
		HTTPCheckList: HTTPChecks{
			&HTTPCheck{
				Type:   "send",
				Method: "POST",
				URI:    "/api",
				CheckHeaders: []*ReturnHeader{
					{Name: &nameVal, Fmt: &fmtVal},
				},
			},
		},
	}

	model := BackendSpecToModel(spec)

	expected := &haproxy_models.Backend{
		BackendBase: haproxy_models.BackendBase{
			Name:     "backend5",
			Balance:  &haproxy_models.Balance{Algorithm: &alg},
			AdvCheck: "tcp-check",
		},
		Servers: map[string]haproxy_models.Server{
			"srvA": {
				Name:    "srvA",
				Address: "192.168.1.1",
				Port:    &port,
				ID:      &id,
			},
		},
		HTTPCheckList: []*haproxy_models.HTTPCheck{
			{
				Type:   "send",
				Method: "POST",
				URI:    "/api",
				CheckHeaders: []*haproxy_models.ReturnHeader{
					{Name: &nameVal, Fmt: &fmtVal},
				},
			},
		},
	}

	if !reflect.DeepEqual(model, expected) {
		t.Errorf("expected model:\n%#v\ngot:\n%#v", expected, model)
	}
}
