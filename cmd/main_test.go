package main

import (
	"os"
	"testing"
)

func TestGetWatchNamespace(t *testing.T) {
	const envVar = "WATCH_NAMESPACE"
	expected := "test-namespace"
	os.Setenv(envVar, expected)
	defer os.Unsetenv(envVar)

	ns, err := getWatchNamespace()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if ns != expected {
		t.Errorf("expected %q, got %q", expected, ns)
	}
}

func TestGetWatchNamespace_NotSet(t *testing.T) {
	const envVar = "WATCH_NAMESPACE"
	os.Unsetenv(envVar)

	_, err := getWatchNamespace()
	if err == nil {
		t.Error("expected error when WATCH_NAMESPACE is not set")
	}
}

func TestGetWatchLabel(t *testing.T) {
	const envVar = "WATCH_LABEL"
	expected := "app=test"
	os.Setenv(envVar, expected)
	defer os.Unsetenv(envVar)

	label, err := getWatchLabel()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if label != expected {
		t.Errorf("expected %q, got %q", expected, label)
	}
}

func TestGetWatchLabel_NotSet(t *testing.T) {
	const envVar = "WATCH_LABEL"
	os.Unsetenv(envVar)

	_, err := getWatchLabel()
	if err == nil {
		t.Error("expected error when WATCH_LABEL is not set")
	}
}

func TestSplitAndFilterEmpty(t *testing.T) {
	input := "a,,b,c,,"
	expected := []string{"a", "b", "c"}
	result := splitAndFilterEmpty(input, ",")
	if len(result) != len(expected) {
		t.Fatalf("expected %d elements, got %d", len(expected), len(result))
	}
	for i, v := range expected {
		if result[i] != v {
			t.Errorf("expected %q at index %d, got %q", v, i, result[i])
		}
	}
}

func TestGetHAProxyURL(t *testing.T) {
	const envVar = "HAPROXY_API_URL"
	expected := "http://haproxy.local"
	os.Setenv(envVar, expected)
	defer os.Unsetenv(envVar)

	val, err := getHAProxyURL()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if val != expected {
		t.Errorf("expected %q, got %q", expected, val)
	}
}

func TestGetHAProxyURL_NotSet(t *testing.T) {
	const envVar = "HAPROXY_API_URL"
	os.Unsetenv(envVar)

	_, err := getHAProxyURL()
	if err == nil {
		t.Error("expected error when HAPROXY_API_URL is not set")
	}
}

func TestGetHAProxyUser(t *testing.T) {
	const envVar = "HAPROXY_API_USER"
	expected := "admin"
	os.Setenv(envVar, expected)
	defer os.Unsetenv(envVar)

	val, err := getHAProxyUser()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if val != expected {
		t.Errorf("expected %q, got %q", expected, val)
	}
}

func TestGetHAProxyUser_NotSet(t *testing.T) {
	const envVar = "HAPROXY_API_USER"
	os.Unsetenv(envVar)

	_, err := getHAProxyUser()
	if err == nil {
		t.Error("expected error when HAPROXY_API_USER is not set")
	}
}

func TestGetHAProxyPass(t *testing.T) {
	const envVar = "HAPROXY_API_PASS"
	expected := "secret"
	os.Setenv(envVar, expected)
	defer os.Unsetenv(envVar)

	val, err := getHAProxyPass()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if val != expected {
		t.Errorf("expected %q, got %q", expected, val)
	}
}

func TestGetHAProxyPass_NotSet(t *testing.T) {
	const envVar = "HAPROXY_API_PASS"
	os.Unsetenv(envVar)

	_, err := getHAProxyPass()
	if err == nil {
		t.Error("expected error when HAPROXY_API_PASS is not set")
	}
}
