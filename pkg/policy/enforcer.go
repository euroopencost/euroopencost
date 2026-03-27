package policy

import (
	"fmt"
	"github.com/euroopencost/euroopencost/internal/models"
)

// Policy defines a rule that must be followed.
type Policy struct {
	ID          string
	Name        string
	Description string
	Check       func(resources []models.Resource) error
}

// Enforcer runs policies against a set of resources.
type Enforcer struct {
	Policies []Policy
}

// NewEnforcer creates a new Enforcer with default policies.
func NewEnforcer() *Enforcer {
	return &Enforcer{
		Policies: []Policy{
			{
				ID:          "POL-001",
				Name:        "Max Monthly Cost",
				Description: "Ensures total monthly cost does not exceed 500€",
				Check: func(resources []models.Resource) error {
					var total float64
					for _, r := range resources {
						total += r.MonthlyPrice()
					}
					if total > 500 {
						return fmt.Errorf("total monthly cost %.2f€ exceeds limit of 500.00€", total)
					}
					return nil
				},
			},
			{
				ID:          "POL-002",
				Name:        "Sovereign Provider Only",
				Description: "Blocks non-EU hyperscalers (AWS, Azure, GCP)",
				Check: func(resources []models.Resource) error {
					for _, r := range resources {
						// Simple check based on provider name
						// In reality, this would be more sophisticated
						if r.Provider == "aws" || r.Provider == "azure" || r.Provider == "google" {
							return fmt.Errorf("resource %s uses non-sovereign provider %s", r.Name, r.Provider)
						}
					}
					return nil
				},
			},
		},
	}
}

// Validate checks all policies against the resources.
func (e *Enforcer) Validate(resources []models.Resource) []error {
	var errs []error
	for _, p := range e.Policies {
		if err := p.Check(resources); err != nil {
			errs = append(errs, fmt.Errorf("[%s] %s: %w", p.ID, p.Name, err))
		}
	}
	return errs
}
