package renderer

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/euroopencost/euroopencost/internal/models"
)

// JSONResource ist die JSON-Darstellung einer einzelnen Ressource.
type JSONResource struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	ServiceName  string `json:"service_name"`
	Flavor       string `json:"flavor"`
	DisplayType  string `json:"display_type"`
	HourlyPrice  string `json:"hourly_price"`
	MonthlyPrice string `json:"monthly_price"`
}

// JSONTotal ist die JSON-Darstellung der Gesamtsumme.
type JSONTotal struct {
	HourlyPrice  string `json:"hourly_price"`
	MonthlyPrice string `json:"monthly_price"`
}

// JSONOutput ist die vollständige JSON-Ausgabe.
type JSONOutput struct {
	Customer  string         `json:"customer,omitempty"`
	Resources []JSONResource `json:"resources"`
	Total     JSONTotal      `json:"total"`
}

// JSONRenderer gibt die Ressourcen als JSON aus.
type JSONRenderer struct {
	customer string
}

// NewJSONRenderer erstellt einen neuen JSONRenderer.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

// Name gibt den Namen des Renderers zurück.
func (r *JSONRenderer) Name() string {
	return "json"
}

// SetCustomer setzt den Kundennamen für den Report.
func (r *JSONRenderer) SetCustomer(name string) {
	r.customer = name
}

// Render gibt alle Ressourcen und die Gesamtsumme als JSON aus.
func (r *JSONRenderer) Render(resources []models.Resource, total models.Total) error {
	jsonResources := make([]JSONResource, 0, len(resources))
	for _, res := range resources {
		jsonResources = append(jsonResources, JSONResource{
			Name:         res.Name,
			Type:         res.Type,
			ServiceName:  res.ServiceName,
			Flavor:       res.Flavor,
			DisplayType:  res.DisplayType(),
			HourlyPrice:  fmt.Sprintf("%.4f", res.HourlyPrice),
			MonthlyPrice: fmt.Sprintf("%.2f", res.MonthlyPrice()),
		})
	}

	output := JSONOutput{
		Customer:  r.customer,
		Resources: jsonResources,
		Total: JSONTotal{
			HourlyPrice:  fmt.Sprintf("%.4f", total.HourlyPrice),
			MonthlyPrice: fmt.Sprintf("%.2f", total.MonthlyPrice),
		},
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("JSON ausgeben fehlgeschlagen: %w", err)
	}
	return nil
}
