package webhook

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"policy-agent/internal/agent"
	"policy-agent/pkg/types"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

// Server is the admission webhook server
type Server struct {
	orchestrator *agent.Orchestrator
	port         int
	certFile     string
	keyFile      string
	server       *http.Server
	decoder      runtime.Decoder
}

// Config holds webhook server configuration
type Config struct {
	Port     int
	CertFile string
	KeyFile  string
}

// NewServer creates a new webhook server
func NewServer(orchestrator *agent.Orchestrator, cfg *Config) *Server {
	scheme := runtime.NewScheme()
	codecs := serializer.NewCodecFactory(scheme)

	return &Server{
		orchestrator: orchestrator,
		port:         cfg.Port,
		certFile:     cfg.CertFile,
		keyFile:      cfg.KeyFile,
		decoder:      codecs.UniversalDeserializer(),
	}
}

// Start starts the webhook server
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/validate", s.handleValidate)
	mux.HandleFunc("/mutate", s.handleMutate)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReady)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Load TLS configuration
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	s.server.TLSConfig = tlsConfig

	fmt.Printf("🚀 Starting webhook server on port %d...\n", s.port)
	fmt.Printf("   TLS Cert: %s\n", s.certFile)
	fmt.Printf("   TLS Key: %s\n", s.keyFile)

	return s.server.ListenAndServeTLS(s.certFile, s.keyFile)
}

// Stop stops the webhook server
func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleValidate handles admission validation requests
func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Decode admission review
	var admissionReview admissionv1.AdmissionReview
	if _, _, err := s.decoder.Decode(body, nil, &admissionReview); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode admission review: %v", err), http.StatusBadRequest)
		return
	}

	if admissionReview.Request == nil {
		http.Error(w, "admission review request is nil", http.StatusBadRequest)
		return
	}

	// Process admission request
	response := s.validateAdmission(r.Context(), admissionReview.Request)

	// Create admission review response
	admissionReview.Response = response
	admissionReview.Response.UID = admissionReview.Request.UID

	// Calculate duration
	duration := time.Since(startTime)
	fmt.Printf("[%s] Validation: %s/%s (%s) - allowed=%v, duration=%v\n",
		time.Now().Format("15:04:05"),
		admissionReview.Request.Kind.Kind,
		admissionReview.Request.Name,
		admissionReview.Request.Namespace,
		response.Allowed,
		duration,
	)

	// Write response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&admissionReview); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// validateAdmission validates an admission request
func (s *Server) validateAdmission(ctx context.Context, request *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	// Extract object
	obj := request.Object.Raw

	// Determine resource type
	resourceType := request.Kind.Kind

	// Prepare validation request
	validateReq := &agent.ValidateRequest{
		Raw:          obj,
		Format:       types.FormatJSON,
		ResourceType: resourceType,
		AutoFix:      false,
	}

	// Validate
	resp, err := s.orchestrator.Validate(ctx, validateReq)
	if err != nil {
		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Status:  "Failure",
				Message: fmt.Sprintf("Validation error: %v", err),
				Code:    http.StatusInternalServerError,
			},
		}
	}

	// Check for violations
	if len(resp.Result.Violations) > 0 {
		message := s.formatViolations(resp.Result)

		return &admissionv1.AdmissionResponse{
			Allowed: false,
			Result: &metav1.Status{
				Status:  "Failure",
				Message: message,
				Reason:  metav1.StatusReasonInvalid,
				Code:    http.StatusForbidden,
			},
		}
	}

	// Allow with warnings
	var warnings []string
	for _, w := range resp.Result.Warnings {
		warnings = append(warnings, fmt.Sprintf("[%s] %s: %s", w.Severity, w.Policy, w.Message))
	}

	return &admissionv1.AdmissionResponse{
		Allowed:  true,
		Warnings: warnings,
	}
}

// handleMutate handles admission mutation requests
func (s *Server) handleMutate(w http.ResponseWriter, r *http.Request) {
	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Decode admission review
	var admissionReview admissionv1.AdmissionReview
	if _, _, err := s.decoder.Decode(body, nil, &admissionReview); err != nil {
		http.Error(w, fmt.Sprintf("failed to decode admission review: %v", err), http.StatusBadRequest)
		return
	}

	if admissionReview.Request == nil {
		http.Error(w, "admission review request is nil", http.StatusBadRequest)
		return
	}

	// For now, just allow without mutation
	// TODO: Implement auto-fix mutation
	response := &admissionv1.AdmissionResponse{
		Allowed: true,
		UID:     admissionReview.Request.UID,
	}

	admissionReview.Response = response

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&admissionReview); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
		return
	}
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleReady handles readiness check requests
func (s *Server) handleReady(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Ready"))
}

// formatViolations formats violations into a user-friendly message
func (s *Server) formatViolations(result *types.ValidationResult) string {
	message := fmt.Sprintf("Policy validation failed with %d violation(s):\n\n", len(result.Violations))

	for i, v := range result.Violations {
		message += fmt.Sprintf("%d. [%s] %s\n", i+1, v.Severity, v.Policy)
		message += fmt.Sprintf("   %s\n", v.Message)

		if v.Field != "" {
			message += fmt.Sprintf("   Field: %s\n", v.Field)
		}

		if v.Remediation != nil && v.Remediation.Suggestion != "" {
			message += fmt.Sprintf("   Suggestion: %s\n", v.Remediation.Suggestion)
		}

		message += "\n"
	}

	return message
}
