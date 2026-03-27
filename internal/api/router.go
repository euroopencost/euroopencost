package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/euroopencost/euroopencost/internal/parser"
	"github.com/euroopencost/euroopencost/internal/pricing"
	"github.com/euroopencost/euroopencost/internal/pricing/hetzner"
	"github.com/euroopencost/euroopencost/internal/pricing/ionos"
	"github.com/euroopencost/euroopencost/internal/pricing/stackit"
	"github.com/euroopencost/euroopencost/internal/scoring"
	"github.com/euroopencost/euroopencost/pkg/policy"
)

// Router handles multi-tenant routing and API endpoints.
type Router struct {
	BaseDomain string
	StaticDir  string
}

// apiResource is the JSON-serializable version of models.Resource with
// snake_case fields and computed MonthlyPrice / DisplayType.
type apiResource struct {
	Provider     string  `json:"provider"`
	Name         string  `json:"name"`
	Type         string  `json:"type"`
	ServiceName  string  `json:"service_name"`
	Flavor       string  `json:"flavor"`
	HourlyPrice  float64 `json:"hourly_price"`
	MonthlyPrice float64 `json:"monthly_price"`
	DisplayType  string  `json:"display_type"`
}

// ServeHTTP implements the http.Handler interface.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Enable CORS for development
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if req.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// API Routes
	if strings.HasPrefix(req.URL.Path, "/api/v1/") {
		r.handleAPI(w, req)
		return
	}

	// Multi-tenant check for static content
	tenant := r.extractTenant(req.Host)
	if tenant != "" {
		// Logic for tenant-specific content (later)
	}

	// Static File Server
	if r.StaticDir != "" {
		http.FileServer(http.Dir(r.StaticDir)).ServeHTTP(w, req)
		return
	}

	// Default response
	fmt.Fprintf(w, "Welcome to EuroOpenCost (eucost) - The Sovereign Cloud Engine\n")
}

func (r *Router) handleAPI(w http.ResponseWriter, req *http.Request) {
	switch req.URL.Path {
	case "/api/v1/analyze":
		r.handleAnalyze(w, req)
	case "/api/v1/health":
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	case "/api/v1/status":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"version":"0.1.0","tier":"pro","status":"ok"}`))
	default:
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

func (r *Router) handleAnalyze(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}

	// 1. Parse Plan
	p := parser.NewParser()
	resources, err := p.ParseReader(bytes.NewReader(body))
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse plan: %v", err), http.StatusBadRequest)
		return
	}

	// 2. Calculate Costs
	calc := pricing.NewCalculator(pricing.NewClient(), hetzner.NewClient(), stackit.NewClient(), ionos.NewClient())
	resources, total, err := calc.Calculate(resources)
	if err != nil {
		http.Error(w, fmt.Sprintf("Price calculation failed: %v", err), http.StatusInternalServerError)
		return
	}

	// 3. Calculate Sovereign Score
	scoreInfo := scoring.CalculateSovereignScore(resources, total)

	// 4. Check Policies
	enforcer := policy.NewEnforcer()
	policyErrs := enforcer.Validate(resources)
	var alerts []string
	for _, pErr := range policyErrs {
		alerts = append(alerts, pErr.Error())
	}

	// Convert resources to API-friendly format with computed fields
	apiResources := make([]apiResource, len(resources))
	for i, r := range resources {
		apiResources[i] = apiResource{
			Provider:     r.Provider,
			Name:         r.Name,
			Type:         r.Type,
			ServiceName:  r.ServiceName,
			Flavor:       r.Flavor,
			HourlyPrice:  r.HourlyPrice,
			MonthlyPrice: r.MonthlyPrice(),
			DisplayType:  r.DisplayType(),
		}
	}

	// Build Response
	resp := map[string]any{
		"resources": apiResources,
		"score":     scoreInfo.Score,
		"scoreInfo": scoreInfo,
		"alerts":    alerts,
		"totals": map[string]float64{
			"hourly":  total.HourlyPrice,
			"monthly": total.MonthlyPrice,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (r *Router) extractTenant(host string) string {
	if !strings.HasSuffix(host, r.BaseDomain) {
		return ""
	}

	prefix := strings.TrimSuffix(host, r.BaseDomain)
	if prefix == "" {
		return ""
	}

	tenant := strings.TrimSuffix(prefix, ".")
	parts := strings.Split(tenant, ".")
	return parts[len(parts)-1]
}

// Start launches the API server.
func (r *Router) Start(addr string) error {
	fmt.Printf("Starting EuroOpenCost API Server on %s...\n", addr)
	return http.ListenAndServe(addr, r)
}
