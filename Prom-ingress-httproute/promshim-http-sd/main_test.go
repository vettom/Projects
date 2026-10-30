package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestHandleIndex(t *testing.T) {
	scheme := runtime.NewScheme()
	route := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "HTTPRoute",
			"metadata": map[string]interface{}{
				"name":      "app-route",
				"namespace": "prod",
			},
			"spec": map[string]interface{}{
				"hostnames": []interface{}{
					"app.example.com",
				},
			},
		},
	}
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-ingress",
			Namespace: "default",
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{Host: "web.example.com"},
			},
		},
	}
	gvrToListKind := map[schema.GroupVersionResource]string{
		httpRouteGVR: "HTTPRouteList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, route)
	server := NewServer(fake.NewSimpleClientset(ingress), dynamicClient)
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	expectedSnippets := []string{
		"/ingress-sd",
		"/httproutes-sd",
		"<table>",
		"<td>default</td>",
		"<td>web-ingress</td>",
		`<a href="https://web.example.com" target="_blank" rel="noopener noreferrer">web.example.com</a>`,
		"<td>prod</td>",
		"<td>app-route</td>",
		`<a href="https://app.example.com" target="_blank" rel="noopener noreferrer">app.example.com</a>`,
	}
	for _, snippet := range expectedSnippets {
		if !contains(body, snippet) {
			t.Fatalf("expected index body to contain %q, got: %s", snippet, body)
		}
	}
}

func TestHandleIndexEmptyRoutes(t *testing.T) {
	scheme := runtime.NewScheme()
	gvrToListKind := map[schema.GroupVersionResource]string{
		httpRouteGVR: "HTTPRouteList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)
	server := NewServer(fake.NewSimpleClientset(), dynamicClient)
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	for _, expected := range []string{
		"No routes found:",
		"No Ingress routes found.",
		"No HTTP routes found.",
	} {
		if !contains(body, expected) {
			t.Fatalf("expected empty routes message %q, got: %s", expected, body)
		}
	}
}

func TestHandleIndexCRDNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	gvrToListKind := map[schema.GroupVersionResource]string{
		httpRouteGVR: "HTTPRouteList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)
	dynamicClient.PrependReactor("list", "httproutes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "gateway.networking.k8s.io", Resource: "httproutes"}, "")
	})

	k8sClient := fake.NewSimpleClientset()
	k8sClient.PrependReactor("list", "ingresses", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "networking.k8s.io", Resource: "ingresses"}, "")
	})

	server := NewServer(k8sClient, dynamicClient)
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if contains(body, "Error fetching") {
		t.Fatalf("expected no error message when resources are not configured, got: %s", body)
	}
	if !contains(body, "No routes found:") {
		t.Fatalf("expected 'No routes found:' message, got: %s", body)
	}
}

func TestHandleHealthz(t *testing.T) {
	server := NewServer(fake.NewSimpleClientset(), dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()))
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res["status"] != "ok" {
		t.Fatalf("expected status ok, got %v", res["status"])
	}
}

func TestGetIngressTargets(t *testing.T) {
	ingress1 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress-1",
			Namespace: "test-ns-1",
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "app1.example.com",
				},
				{
					Host: "app2.example.com",
				},
				{
					Host: "", // should be skipped
				},
			},
		},
	}
	ingress2 := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress-2",
			Namespace: "test-ns-2",
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: "api.example.com",
				},
			},
		},
	}

	k8sClient := fake.NewSimpleClientset(ingress1, ingress2)
	server := NewServer(k8sClient, dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()))

	targets, err := server.GetIngressTargets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 3 {
		t.Fatalf("expected 3 targets, got %d", len(targets))
	}

	expectedHosts := map[string]string{
		"app1.example.com": "test-ns-1",
		"app2.example.com": "test-ns-1",
		"api.example.com":  "test-ns-2",
	}

	for _, tg := range targets {
		if len(tg.Targets) != 1 {
			t.Fatalf("expected 1 host per target group, got %d", len(tg.Targets))
		}
		host := tg.Targets[0]
		expectedNs, ok := expectedHosts[host]
		if !ok {
			t.Fatalf("unexpected host %s", host)
		}
		if tg.Labels["namespace"] != expectedNs {
			t.Fatalf("expected namespace %s for host %s, got %s", expectedNs, host, tg.Labels["namespace"])
		}
	}
}

func TestGetIngressTargetsEmpty(t *testing.T) {
	k8sClient := fake.NewSimpleClientset()
	server := NewServer(k8sClient, dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()))

	targets, err := server.GetIngressTargets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if targets == nil || len(targets) != 0 {
		t.Fatalf("expected empty non-nil slice, got %v", targets)
	}

	// Verify JSON marshaling produces "[]"
	bytes, err := json.Marshal(targets)
	if err != nil {
		t.Fatalf("failed to marshal targets: %v", err)
	}
	if string(bytes) != "[]" {
		t.Fatalf("expected JSON '[]', got '%s'", string(bytes))
	}
}

func TestGetHTTPRouteTargets(t *testing.T) {
	scheme := runtime.NewScheme()

	route1 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "HTTPRoute",
			"metadata": map[string]interface{}{
				"name":      "route-1",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"hostnames": []interface{}{
					"svc1.example.com",
					"svc2.example.com",
				},
			},
		},
	}
	route2 := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "HTTPRoute",
			"metadata": map[string]interface{}{
				"name":      "route-2",
				"namespace": "test",
			},
			"spec": map[string]interface{}{
				"hostnames": []interface{}{
					"svc2.example.com", // duplicate
					"svc3.example.com",
				},
			},
		},
	}

	gvrToListKind := map[schema.GroupVersionResource]string{
		httpRouteGVR: "HTTPRouteList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, route1, route2)
	server := NewServer(fake.NewSimpleClientset(), dynamicClient)

	targets, err := server.GetHTTPRouteTargets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(targets) != 1 {
		t.Fatalf("expected 1 target group, got %d", len(targets))
	}

	tg := targets[0]
	if len(tg.Targets) != 3 {
		t.Fatalf("expected 3 unique targets, got %d: %v", len(tg.Targets), tg.Targets)
	}

	expected := []string{"svc1.example.com", "svc2.example.com", "svc3.example.com"}
	for i, exp := range expected {
		if tg.Targets[i] != exp {
			t.Fatalf("expected target %d to be %s, got %s", i, exp, tg.Targets[i])
		}
	}

	if tg.Labels["__meta_application"] != "httproutes" {
		t.Fatalf("unexpected label __meta_application: %s", tg.Labels["__meta_application"])
	}
	if tg.Labels["__meta_source"] != "prom_shim_http_routes" {
		t.Fatalf("unexpected label __meta_source: %s", tg.Labels["__meta_source"])
	}
	if tg.Labels["__meta_origin"] != "eks_cluster" {
		t.Fatalf("unexpected label __meta_origin: %s", tg.Labels["__meta_origin"])
	}
}

func TestGetHTTPRouteTargetsEmpty(t *testing.T) {
	scheme := runtime.NewScheme()
	gvrToListKind := map[schema.GroupVersionResource]string{
		httpRouteGVR: "HTTPRouteList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)
	server := NewServer(fake.NewSimpleClientset(), dynamicClient)

	targets, err := server.GetHTTPRouteTargets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if targets == nil || len(targets) != 0 {
		t.Fatalf("expected empty non-nil slice, got %v", targets)
	}

	bytes, err := json.Marshal(targets)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	expectedJSON := `[]`
	if string(bytes) != expectedJSON {
		t.Fatalf("expected %s, got %s", expectedJSON, string(bytes))
	}
}

func TestHandleHTTPRouteSDCRDNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	gvrToListKind := map[schema.GroupVersionResource]string{
		httpRouteGVR: "HTTPRouteList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind)
	dynamicClient.PrependReactor("list", "httproutes", func(action k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewNotFound(schema.GroupResource{Group: "gateway.networking.k8s.io", Resource: "httproutes"}, "")
	})
	server := NewServer(fake.NewSimpleClientset(), dynamicClient)
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/httproutes-sd", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("expected application/json; charset=utf-8 Content-Type, got %s", ct)
	}
	if got := rec.Body.String(); got != "[]\n" {
		t.Fatalf("expected empty JSON array '[]\\n', got %q", got)
	}
}

func TestHandleIngressSDHTTP(t *testing.T) {
	ingress := &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ingress",
			Namespace: "default",
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{Host: "web.example.com"},
			},
		},
	}
	server := NewServer(fake.NewSimpleClientset(ingress), dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()))
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/ingress-sd", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("expected application/json; charset=utf-8 Content-Type, got %s", ct)
	}

	var targets []TargetGroup
	if err := json.Unmarshal(rec.Body.Bytes(), &targets); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(targets) != 1 || targets[0].Targets[0] != "web.example.com" {
		t.Fatalf("unexpected targets: %v", targets)
	}
}

func TestHandleHTTPRouteSDHTTP(t *testing.T) {
	scheme := runtime.NewScheme()
	route := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "HTTPRoute",
			"metadata": map[string]interface{}{
				"name":      "route-1",
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"hostnames": []interface{}{
					"gateway.example.com",
				},
			},
		},
	}
	gvrToListKind := map[schema.GroupVersionResource]string{
		httpRouteGVR: "HTTPRouteList",
	}
	dynamicClient := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(scheme, gvrToListKind, route)
	server := NewServer(fake.NewSimpleClientset(), dynamicClient)
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/httproutes-sd", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var targets []TargetGroup
	if err := json.Unmarshal(rec.Body.Bytes(), &targets); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(targets) != 1 || len(targets[0].Targets) != 1 || targets[0].Targets[0] != "gateway.example.com" {
		t.Fatalf("unexpected targets: %v", targets)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	server := NewServer(fake.NewSimpleClientset(), dynamicfake.NewSimpleDynamicClient(runtime.NewScheme()))
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	endpoints := []string{"/", "/ingress-sd", "/httproutes-sd", "/healthz"}
	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodPost, ep, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405 Method Not Allowed for POST %s, got %d", ep, rec.Code)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

