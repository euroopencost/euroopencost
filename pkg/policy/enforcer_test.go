package policy

import (
	"strings"
	"testing"

	"github.com/euroopencost/euroopencost/internal/models"
)

func TestValidate_AllPass(t *testing.T) {
	e := NewEnforcer()
	resources := []models.Resource{
		{Provider: "hetzner", HourlyPrice: 0.005}, // ~3.6€/month
	}
	errs := e.Validate(resources)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestValidate_ExceedsCostLimit(t *testing.T) {
	e := NewEnforcer()
	// 1.0€/h → 720€/month — exceeds 500€ POL-001 limit
	resources := []models.Resource{
		{Provider: "hetzner", HourlyPrice: 1.0},
	}
	errs := e.Validate(resources)
	if len(errs) == 0 {
		t.Fatal("expected POL-001 cost limit error, got none")
	}
	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "POL-001") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected [POL-001] in errors, got: %v", errs)
	}
}

func TestValidate_NonSovereignProvider_AWS(t *testing.T) {
	e := NewEnforcer()
	resources := []models.Resource{
		{Provider: "aws", Name: "prod-server", HourlyPrice: 0.01},
	}
	errs := e.Validate(resources)
	if len(errs) == 0 {
		t.Fatal("expected POL-002 sovereign provider error, got none")
	}
	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "POL-002") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected [POL-002] in errors, got: %v", errs)
	}
}

func TestValidate_NonSovereignProvider_Azure(t *testing.T) {
	e := NewEnforcer()
	resources := []models.Resource{
		{Provider: "azure", Name: "vm-01", HourlyPrice: 0.01},
	}
	errs := e.Validate(resources)
	found := false
	for _, err := range errs {
		if strings.Contains(err.Error(), "POL-002") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected [POL-002] for azure, got: %v", errs)
	}
}

func TestValidate_BothViolations(t *testing.T) {
	e := NewEnforcer()
	// aws + 2.0€/h (>500€/month) → both POL-001 and POL-002
	resources := []models.Resource{
		{Provider: "aws", Name: "big-server", HourlyPrice: 2.0},
	}
	errs := e.Validate(resources)
	if len(errs) != 2 {
		t.Errorf("expected 2 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidate_EUSovereignProviderOK(t *testing.T) {
	e := NewEnforcer()
	resources := []models.Resource{
		{Provider: "otc", HourlyPrice: 0.01},
		{Provider: "stackit", HourlyPrice: 0.01},
		{Provider: "ionos", HourlyPrice: 0.01},
	}
	errs := e.Validate(resources)
	if len(errs) != 0 {
		t.Errorf("expected no errors for EU-sovereign providers, got: %v", errs)
	}
}
