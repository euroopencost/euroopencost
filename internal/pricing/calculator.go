package pricing

import (
	"fmt"
	"os"

	"github.com/euroopencost/euroopencost/internal/models"
	"github.com/euroopencost/euroopencost/internal/pricing/hetzner"
	"github.com/euroopencost/euroopencost/internal/pricing/ionos"
	"github.com/euroopencost/euroopencost/internal/pricing/stackit"
)

// Calculator berechnet die Preise für alle unterstützten Cloud-Anbieter.
type Calculator struct {
	otcClient     *Client
	hetznerClient *hetzner.Client
	stackitClient *stackit.Client
	ionosClient   *ionos.Client
}

// NewCalculator erstellt einen neuen Calculator mit allen Provider-Clients.
func NewCalculator(otc *Client, hcloud *hetzner.Client, sk *stackit.Client, io *ionos.Client) *Calculator {
	return &Calculator{
		otcClient:     otc,
		hetznerClient: hcloud,
		stackitClient: sk,
		ionosClient:   io,
	}
}

// Calculate berechnet die Preise für alle Ressourcen und gibt eine Gesamtsumme zurück.
func (c *Calculator) Calculate(resources []models.Resource) ([]models.Resource, models.Total, error) {
	var total models.Total

	for i, res := range resources {
		var hourlyPrice float64
		var err error

		switch res.Provider {
		case "hetzner":
			hourlyPrice, err = c.hetznerClient.GetPriceForResource(res)
			if err != nil {
				return nil, models.Total{}, fmt.Errorf("Hetzner-Preis für '%s' fehlgeschlagen: %w", res.Name, err)
			}

		case "stackit":
			hourlyPrice, err = c.stackitClient.GetPriceForResource(res)
			if err != nil {
				return nil, models.Total{}, fmt.Errorf("STACKIT-Preis für '%s' fehlgeschlagen: %w", res.Name, err)
			}

		case "ionos":
			hourlyPrice, err = c.ionosClient.GetPriceForResource(res)
			if err != nil {
				return nil, models.Total{}, fmt.Errorf("IONOS-Preis für '%s' fehlgeschlagen: %w", res.Name, err)
			}

		default: // "otc" oder leer → OTC API
			hourlyPrice, err = c.calculateOTCPrice(res)
			if err != nil {
				return nil, models.Total{}, err
			}
		}

		resources[i].HourlyPrice = hourlyPrice
		total.HourlyPrice += hourlyPrice
		total.MonthlyPrice += resources[i].MonthlyPrice()
	}

	return resources, total, nil
}

// calculateOTCPrice berechnet den Preis für eine OTC-Ressource.
func (c *Calculator) calculateOTCPrice(res models.Resource) (float64, error) {
	prices, err := c.otcClient.GetPrices(res.ServiceName)
	if err != nil {
		return 0, fmt.Errorf("OTC-Preis für '%s' (%s) fehlgeschlagen: %w", res.Name, res.ServiceName, err)
	}

	// Kostenlose Ressource (nil von API)
	if prices == nil {
		return 0, nil
	}

	// APIFlavor bevorzugen, sonst Flavor als Fallback
	lookupFlavor := res.APIFlavor
	if lookupFlavor == "" {
		lookupFlavor = res.Flavor
	}

	entry, found := findPriceEntry(prices, lookupFlavor)
	if !found {
		fmt.Fprintf(os.Stderr, "Warnung: Kein OTC-Preis für Flavor '%s' (Service: %s)\n", lookupFlavor, res.ServiceName)
		return 0, nil
	}

	price, err := ParsePrice(entry.PriceAmount)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warnung: Preis parsen fehlgeschlagen für '%s': %v\n", lookupFlavor, err)
		return 0, nil
	}

	// Mengenbasierte Preise: Einheit "GB" = pro GB pro Monat → Stundenpreis
	if entry.Unit == "GB" && res.Quantity > 0 {
		return (price * res.Quantity) / (24 * 30), nil
	}

	return price, nil
}

// findPriceEntry sucht den Preiseintrag für einen Flavor in der Preisliste.
func findPriceEntry(prices []PriceEntry, flavor string) (PriceEntry, bool) {
	for _, entry := range prices {
		if entry.OpiFlavour == flavor {
			return entry, true
		}
	}
	return PriceEntry{}, false
}
