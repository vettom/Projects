// Prometheus shim application to expose Kubernetes Ingress and HTTPRoute resources as Prometheus service discovery targets.
// This application runs inside a Kubernetes cluster and uses the in-cluster configuration to access the Kubernetes API.
// It retrieves all Ingress and Gateway API HTTPRoute resources across all namespaces and formats them as Prometheus service discovery targets.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// isResourceNotConfigured checks whether an error indicates that the Kubernetes
// resource or CRD is not installed or not found on the cluster.
func isResourceNotConfigured(err error) bool {
	if err == nil {
		return false
	}
	if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "could not find the requested resource") ||
		strings.Contains(msg, "no matches for")
}

// TargetGroup represents a Prometheus HTTP service discovery target group.
type TargetGroup struct {
	Targets []string          `json:"targets"`
	Labels  map[string]string `json:"labels"`
}

// HTTPRoute GVR definition
var httpRouteGVR = schema.GroupVersionResource{
	Group:    "gateway.networking.k8s.io",
	Version:  "v1",
	Resource: "httproutes",
}

// Server encapsulates the HTTP handler dependencies.
type Server struct {
	k8sClient     kubernetes.Interface
	dynamicClient dynamic.Interface
}

// NewServer creates a new Server instance.
func NewServer(k8sClient kubernetes.Interface, dynamicClient dynamic.Interface) *Server {
	return &Server{
		k8sClient:     k8sClient,
		dynamicClient: dynamicClient,
	}
}

// GetIngressTargets retrieves all Ingress resources across all namespaces
// and formats them as Prometheus service discovery targets.
func (s *Server) GetIngressTargets(ctx context.Context) ([]TargetGroup, error) {
	targets := make([]TargetGroup, 0)
	ingressList, err := s.k8sClient.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		if isResourceNotConfigured(err) {
			log.Printf("Ingress resource not found or not configured, returning empty targets: %v", err)
			return targets, nil
		}
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	for _, ingress := range ingressList.Items {
		namespace := ingress.Namespace
		name := ingress.Name

		for _, rule := range ingress.Spec.Rules {
			host := rule.Host
			if host == "" {
				continue
			}

			// This section is ignoring the paths in ingress rules.
			targets = append(targets, TargetGroup{
				Targets: []string{host},
				Labels: map[string]string{
					"namespace": namespace,
					"ingress":   name,
				},
			})

			// Uncomment the following section if you want to include paths in the targets
			// if rule.HTTP != nil {
			// 	for _, pathObj := range rule.HTTP.Paths {
			// 		path := pathObj.Path
			// 		if path == "" {
			// 			path = "/"
			// 		}
			// 		url := fmt.Sprintf("https://%s%s", host, path)
			// 		targets = append(targets, TargetGroup{
			// 			Targets: []string{url},
			// 			Labels: map[string]string{
			// 				"namespace": namespace,
			// 				"ingress":   name,
			// 			},
			// 		})
			// 	}
			// }
		}
	}

	return targets, nil
}

// HTTPRouteEntry represents a discovered route hostname entry for display on the home page.
type HTTPRouteEntry struct {
	Namespace string
	Name      string
	Hostname  string
	URL       string
}

// GetIngressEntries retrieves all Kubernetes Ingress resources across all namespaces
// and returns structured entries sorted by namespace, name, and hostname.
func (s *Server) GetIngressEntries(ctx context.Context) ([]HTTPRouteEntry, error) {
	entries := make([]HTTPRouteEntry, 0)
	ingressList, err := s.k8sClient.NetworkingV1().Ingresses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		if isResourceNotConfigured(err) {
			log.Printf("Ingress resource not found or not configured, returning empty entries: %v", err)
			return entries, nil
		}
		return nil, fmt.Errorf("failed to list ingresses: %w", err)
	}

	for _, ingress := range ingressList.Items {
		namespace := ingress.Namespace
		name := ingress.Name

		for _, rule := range ingress.Spec.Rules {
			host := rule.Host
			if host == "" {
				continue
			}
			url := host
			if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
				url = "https://" + host
			}
			entries = append(entries, HTTPRouteEntry{
				Namespace: namespace,
				Name:      name,
				Hostname:  host,
				URL:       url,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Namespace != entries[j].Namespace {
			return entries[i].Namespace < entries[j].Namespace
		}
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Hostname < entries[j].Hostname
	})

	return entries, nil
}

var indexTemplate = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
    <head>
        <meta charset="utf-8">
        <title>Prometheus Ingress and httproutes Service Discovery</title>
        <style>
            body { font-family: sans-serif; margin: 2rem; }
            table { border-collapse: collapse; width: 100%; max-width: 900px; margin-top: 1rem; }
            th, td { border: 1px solid #ccc; padding: 0.5rem 0.75rem; text-align: left; }
            th { background-color: #f4f4f4; }
            tr:nth-child(even) { background-color: #fafafa; }
            .error { color: #b00020; }
            .empty {
                background-color: #f8f9fa;
                border: 1px solid #dee2e6;
                border-left: 4px solid #6c757d;
                padding: 0.75rem 1rem;
                max-width: 900px;
                color: #495057;
            }
        </style>
    </head>
    <body>
        <h1>Prometheus Ingress and httproutes Service Discovery.</h1>
        <h3>Available endpoints:</h3>
        <ul>
            <li><a href="/ingress-sd">/ingress-sd</a> - Get Ingress Service Discovery targets</li>
            <li><a href="/httproutes-sd">/httproutes-sd</a> - Get HTTP Routes Service Discovery targets</li>
            <li><a href="/healthz">/healthz</a> - Health check endpoint</li>
        </ul>

        {{- if and (not .Ingresses) (not .Routes) }}
        <p class="empty"><strong>No routes found:</strong> No Ingress or HTTPRoute resources are currently configured in the cluster.</p>
        {{- end }}

        <h3>Discovered Ingresses:</h3>
        {{- if .IngressError }}
        <p class="error">Error fetching Ingresses: {{ .IngressError }}</p>
        {{- end }}
        {{- if .Ingresses }}
        <table>
            <thead>
                <tr>
                    <th>Namespace</th>
                    <th>Name</th>
                    <th>Hostname</th>
                </tr>
            </thead>
            <tbody>
                {{- range .Ingresses }}
                <tr>
                    <td>{{ .Namespace }}</td>
                    <td>{{ .Name }}</td>
                    <td><a href="{{ .URL }}" target="_blank" rel="noopener noreferrer">{{ .Hostname }}</a></td>
                </tr>
                {{- end }}
            </tbody>
        </table>
        {{- else }}
        <p class="empty">No Ingress routes found.</p>
        {{- end }}

        <h3>Discovered HTTP Routes:</h3>
        {{- if .Error }}
        <p class="error">Error fetching HTTP routes: {{ .Error }}</p>
        {{- end }}
        {{- if .Routes }}
        <table>
            <thead>
                <tr>
                    <th>Namespace</th>
                    <th>Name</th>
                    <th>Hostname</th>
                </tr>
            </thead>
            <tbody>
                {{- range .Routes }}
                <tr>
                    <td>{{ .Namespace }}</td>
                    <td>{{ .Name }}</td>
                    <td><a href="{{ .URL }}" target="_blank" rel="noopener noreferrer">{{ .Hostname }}</a></td>
                </tr>
                {{- end }}
            </tbody>
        </table>
        {{- else }}
        <p class="empty">No HTTP routes found.</p>
        {{- end }}
    </body>
</html>
`))

// GetHTTPRouteEntries retrieves all Gateway API HTTPRoute resources across all namespaces
// and returns structured entries sorted by namespace, name, and hostname.
func (s *Server) GetHTTPRouteEntries(ctx context.Context) ([]HTTPRouteEntry, error) {
	entries := make([]HTTPRouteEntry, 0)
	routeList, err := s.dynamicClient.Resource(httpRouteGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		if isResourceNotConfigured(err) {
			log.Printf("HTTPRoute resource not found or not configured, returning empty entries: %v", err)
			return entries, nil
		}
		return nil, fmt.Errorf("failed to list httproutes: %w", err)
	}

	for _, item := range routeList.Items {
		namespace := item.GetNamespace()
		name := item.GetName()

		spec, ok := item.Object["spec"].(map[string]interface{})
		if !ok {
			continue
		}
		rawHostnames, ok := spec["hostnames"].([]interface{})
		if !ok {
			continue
		}
		for _, raw := range rawHostnames {
			if hostname, ok := raw.(string); ok && hostname != "" {
				url := hostname
				if !strings.HasPrefix(hostname, "http://") && !strings.HasPrefix(hostname, "https://") {
					url = "https://" + hostname
				}
				entries = append(entries, HTTPRouteEntry{
					Namespace: namespace,
					Name:      name,
					Hostname:  hostname,
					URL:       url,
				})
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Namespace != entries[j].Namespace {
			return entries[i].Namespace < entries[j].Namespace
		}
		if entries[i].Name != entries[j].Name {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Hostname < entries[j].Hostname
	})

	return entries, nil
}

// GetHTTPRouteTargets retrieves all Gateway API HTTPRoute resources across all namespaces
// and formats them as Prometheus service discovery targets.
func (s *Server) GetHTTPRouteTargets(ctx context.Context) ([]TargetGroup, error) {
	emptyTargets := make([]TargetGroup, 0)
	routeList, err := s.dynamicClient.Resource(httpRouteGVR).List(ctx, metav1.ListOptions{})
	if err != nil {
		if isResourceNotConfigured(err) {
			log.Printf("HTTPRoute resource not found or not configured, returning empty targets: %v", err)
			return emptyTargets, nil
		}
		return nil, fmt.Errorf("failed to list httproutes: %w", err)
	}

	hostnameSet := make(map[string]struct{})
	for _, item := range routeList.Items {
		spec, ok := item.Object["spec"].(map[string]interface{})
		if !ok {
			continue
		}
		rawHostnames, ok := spec["hostnames"].([]interface{})
		if !ok {
			continue
		}
		for _, raw := range rawHostnames {
			if hostname, ok := raw.(string); ok && hostname != "" {
				hostnameSet[hostname] = struct{}{}
			}
		}
	}

	if len(hostnameSet) == 0 {
		return emptyTargets, nil
	}

	hostnames := make([]string, 0, len(hostnameSet))
	for h := range hostnameSet {
		hostnames = append(hostnames, h)
	}
	sort.Strings(hostnames)

	promData := TargetGroup{
		Targets: hostnames,
		Labels: map[string]string{
			"__meta_application": "httproutes",
			"__meta_source":      "prom_shim_http_routes",
			"__meta_origin":      "eks_cluster",
		},
	}

	return []TargetGroup{promData}, nil
}

// handleIndex serves the landing HTML page with available endpoints and discovered routes.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ingresses, err := s.GetIngressEntries(r.Context())
	var ingressErrMsg string
	if err != nil {
		log.Printf("Error fetching Ingress entries for index page: %v", err)
		ingressErrMsg = err.Error()
	}

	routes, err := s.GetHTTPRouteEntries(r.Context())
	var errMsg string
	if err != nil {
		log.Printf("Error fetching HTTPRoute entries for index page: %v", err)
		errMsg = err.Error()
	}

	data := struct {
		Ingresses    []HTTPRouteEntry
		IngressError string
		Routes       []HTTPRouteEntry
		Error        string
	}{
		Ingresses:    ingresses,
		IngressError: ingressErrMsg,
		Routes:       routes,
		Error:        errMsg,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if err := indexTemplate.Execute(w, data); err != nil {
		log.Printf("Error rendering index template: %v", err)
	}
}

// handleIngressSD serves the Ingress service discovery targets as JSON.
func (s *Server) handleIngressSD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targets, err := s.GetIngressTargets(r.Context())
	if err != nil {
		log.Printf("Error fetching ingress targets, returning empty target list: %v", err)
		writeJSON(w, http.StatusOK, make([]TargetGroup, 0))
		return
	}

	writeJSON(w, http.StatusOK, targets)
}

// handleHTTPRouteSD serves the HTTPRoute service discovery targets as JSON.
func (s *Server) handleHTTPRouteSD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	targets, err := s.GetHTTPRouteTargets(r.Context())
	if err != nil {
		log.Printf("Error fetching HTTPRoute targets, returning empty target list: %v", err)
		writeJSON(w, http.StatusOK, make([]TargetGroup, 0))
		return
	}

	writeJSON(w, http.StatusOK, targets)
}

// handleHealthz serves health check requests.
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// writeJSON writes JSON response with appropriate headers.
func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
	}
}

// RegisterRoutes registers all HTTP handlers to the provided mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/ingress-sd", s.handleIngressSD)
	mux.HandleFunc("/httproutes-sd", s.handleHTTPRouteSD)
	mux.HandleFunc("/healthz", s.handleHealthz)
}

// loggingResponseWriter tracks HTTP status code for logging.
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

// loggingMiddleware logs HTTP request details.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)
		log.Printf("%s %s %s %d %s", r.RemoteAddr, r.Method, r.URL.Path, lrw.statusCode, time.Since(start))
	})
}

// getKubeConfig loads in-cluster Kubernetes configuration if available,
// or falls back to local kubeconfig.
func getKubeConfig() (*rest.Config, error) {
	// First try in-cluster configuration
	config, err := rest.InClusterConfig()
	if err == nil {
		log.Println("Loaded in-cluster Kubernetes configuration")
		return config, nil
	}

	// Fallback to kubeconfig file
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	if kubeconfig != "" {
		if _, err := os.Stat(kubeconfig); err == nil {
			log.Printf("Loaded local kubeconfig from %s", kubeconfig)
			return clientcmd.BuildConfigFromFlags("", kubeconfig)
		}
	}

	return nil, fmt.Errorf("unable to load in-cluster configuration or local kubeconfig: %w", err)
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9113"
	}
	addr := ":" + port

	config, err := getKubeConfig()
	if err != nil {
		log.Fatalf("Failed to initialize Kubernetes configuration: %v", err)
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes clientset: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create dynamic Kubernetes client: %v", err)
	}

	server := NewServer(k8sClient, dynamicClient)
	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle graceful shutdown on OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Starting Prometheus HTTP SD shim server on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Server terminated unexpectedly: %v", err)
		}
	}()

	<-stop
	log.Println("Shutdown signal received, shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server gracefully stopped")
}
