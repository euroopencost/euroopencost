package stackit

import (
	"fmt"
	"os"

	"github.com/euroopencost/euroopencost/internal/models"
)

// machineSpec enthält die Hardware-Spezifikation eines STACKIT Machine Types.
type machineSpec struct {
	VCPU      float64
	RAMGB     float64
	Dedicated bool // true = kein CPU-Overprovisioning (d-Suffix, AMD/Intel Gen1/2)
}

// rateSet enthält die Preisraten pro vCPU und pro GB RAM (EUR/h).
type rateSet struct {
	PerVCPU  float64
	PerGBRAM float64
}

// Client ist der STACKIT Pricing Client.
// STACKIT bietet kein öffentliches REST-API für Preise.
// Preis = vCPU × ratePerVCPU + RAM_GB × ratePerGBRAM
// TODO: Aktuelle Preise unter https://stackit.com/en/prices/cloud prüfen
// und Rates in loadRates() aktualisieren.
type Client struct {
	specs         map[string]machineSpec // machine_type → Spezifikation
	overprovRate  rateSet                // EUR/h für overprovisioned (Gen3, kein d-Suffix)
	dedicatedRate rateSet                // EUR/h für dedicated (Gen1/2, d-Suffix)
	volumePrice   float64               // EUR/GB/Monat (Performance SSD)
}

// NewClient erstellt einen neuen STACKIT Client.
func NewClient() *Client {
	c := &Client{
		specs: make(map[string]machineSpec),
	}
	c.loadSpecs()
	c.loadRates()
	return c
}

// GetPriceForResource gibt den stündlichen Preis für eine STACKIT-Ressource zurück.
func (c *Client) GetPriceForResource(res models.Resource) (float64, error) {
	switch res.ServiceName {
	case "stackit-server":
		spec, ok := c.specs[res.APIFlavor]
		if !ok {
			fmt.Fprintf(os.Stderr, "Warnung: Unbekannter STACKIT Machine-Type '%s'\n", res.APIFlavor)
			return 0, nil
		}
		rates := c.overprovRate
		if spec.Dedicated {
			rates = c.dedicatedRate
		}
		return spec.VCPU*rates.PerVCPU + spec.RAMGB*rates.PerGBRAM, nil

	case "stackit-volume":
		// Preis: EUR/GB/Monat → EUR/h (720 Stunden/Monat)
		if res.Quantity > 0 {
			return (c.volumePrice * res.Quantity) / 720, nil
		}
		return 0, nil

	case "stackit-obs", "stackit-free":
		return 0, nil
	}

	return 0, nil
}

// loadRates lädt die Preisraten für overprovisioned und dedicated Machine Types.
// Schätzwerte, Stand 2025 (netto EUR) — STACKIT veröffentlicht keine Maschinenlesbare Preisliste.
// TODO: Rates unter https://stackit.com/en/prices/cloud verifizieren.
func (c *Client) loadRates() {
	// Overprovisioned (Gen3, Intel, kein d-Suffix): ~Hetzner-shared-Niveau, DE-Cloud-Aufschlag
	c.overprovRate = rateSet{
		PerVCPU:  0.0040, // EUR/h pro vCPU
		PerGBRAM: 0.0020, // EUR/h pro GB RAM
	}
	// Dedicated (Gen1/2, AMD/Intel, d-Suffix): höherer CPU-Garantie-Aufschlag
	c.dedicatedRate = rateSet{
		PerVCPU:  0.0090, // EUR/h pro vCPU
		PerGBRAM: 0.0030, // EUR/h pro GB RAM
	}
	c.volumePrice = 0.0550 // EUR/GB/Monat (Performance SSD)
}

// loadSpecs lädt alle bekannten STACKIT Machine Type Spezifikationen.
// Quelle: https://docs.stackit.cloud/products/compute-engine/server/basics/machine-types/
// Stand: 2025-03
func (c *Client) loadSpecs() {
	// --- Intel Gen3, Overprovisioned (kein d-Suffix) ---
	// t-Serie: tiny (1:1 vCPU:RAM ratio)
	c.specs["t3i.1"] = machineSpec{1, 1, false}
	// s-Serie: standard (1:1 vCPU:RAM)
	c.specs["s3i.2"] = machineSpec{2, 2, false}
	c.specs["s3i.4"] = machineSpec{4, 4, false}
	c.specs["s3i.8"] = machineSpec{8, 8, false}
	c.specs["s3i.16"] = machineSpec{16, 16, false}
	c.specs["s3i.28"] = machineSpec{28, 28, false}
	c.specs["s3i.56"] = machineSpec{56, 56, false}
	c.specs["s3i.112"] = machineSpec{112, 112, false}
	// c-Serie: compute (1:2 vCPU:RAM)
	c.specs["c3i.1"] = machineSpec{1, 2, false}
	c.specs["c3i.2"] = machineSpec{2, 4, false}
	c.specs["c3i.4"] = machineSpec{4, 8, false}
	c.specs["c3i.8"] = machineSpec{8, 16, false}
	c.specs["c3i.16"] = machineSpec{16, 32, false}
	c.specs["c3i.28"] = machineSpec{28, 59, false}
	c.specs["c3i.56"] = machineSpec{56, 118, false}
	c.specs["c3i.112"] = machineSpec{112, 236, false}
	// g-Serie: general (1:4 vCPU:RAM)
	c.specs["g3i.1"] = machineSpec{1, 4, false}
	c.specs["g3i.2"] = machineSpec{2, 8, false}
	c.specs["g3i.4"] = machineSpec{4, 16, false}
	c.specs["g3i.8"] = machineSpec{8, 32, false}
	c.specs["g3i.16"] = machineSpec{16, 59, false}
	c.specs["g3i.28"] = machineSpec{28, 118, false}
	c.specs["g3i.56"] = machineSpec{56, 236, false}
	// m-Serie: memory (1:8 vCPU:RAM)
	c.specs["m3i.1"] = machineSpec{1, 8, false}
	c.specs["m3i.2"] = machineSpec{2, 16, false}
	c.specs["m3i.4"] = machineSpec{4, 32, false}
	c.specs["m3i.8"] = machineSpec{8, 59, false}
	c.specs["m3i.16"] = machineSpec{16, 118, false}
	// b-Serie: big-memory (1:16 vCPU:RAM)
	c.specs["b3i.1"] = machineSpec{1, 16, false}
	c.specs["b3i.2"] = machineSpec{2, 32, false}
	c.specs["b3i.4"] = machineSpec{4, 59, false}
	c.specs["b3i.8"] = machineSpec{8, 118, false}
	c.specs["b3i.16"] = machineSpec{16, 236, false}

	// --- AMD Gen2, Dedicated (d-Suffix) ---
	// c-Serie: compute
	c.specs["c2a.1d"] = machineSpec{1, 2, true}
	c.specs["c2a.2d"] = machineSpec{2, 4, true}
	c.specs["c2a.4d"] = machineSpec{4, 8, true}
	c.specs["c2a.8d"] = machineSpec{8, 16, true}
	c.specs["c2a.16d"] = machineSpec{16, 32, true}
	c.specs["c2a.30d"] = machineSpec{30, 60, true}
	c.specs["c2a.60d"] = machineSpec{60, 120, true}
	c.specs["c2a.120d"] = machineSpec{120, 240, true}
	c.specs["c2a.240d"] = machineSpec{240, 480, true}
	// g-Serie: general
	c.specs["g2a.1d"] = machineSpec{1, 4, true}
	c.specs["g2a.2d"] = machineSpec{2, 8, true}
	c.specs["g2a.4d"] = machineSpec{4, 16, true}
	c.specs["g2a.8d"] = machineSpec{8, 32, true}
	c.specs["g2a.16d"] = machineSpec{16, 60, true}
	c.specs["g2a.30d"] = machineSpec{30, 120, true}
	c.specs["g2a.60d"] = machineSpec{60, 240, true}
	c.specs["g2a.120d"] = machineSpec{120, 480, true}
	// m-Serie: memory
	c.specs["m2a.1d"] = machineSpec{1, 8, true}
	c.specs["m2a.2d"] = machineSpec{2, 16, true}
	c.specs["m2a.4d"] = machineSpec{4, 32, true}
	c.specs["m2a.8d"] = machineSpec{8, 60, true}
	c.specs["m2a.16d"] = machineSpec{16, 120, true}
	c.specs["m2a.30d"] = machineSpec{30, 240, true}
	c.specs["m2a.60d"] = machineSpec{60, 480, true}
	c.specs["m2a.120d"] = machineSpec{120, 960, true}
	// b-Serie: big-memory
	c.specs["b2a.1d"] = machineSpec{1, 16, true}
	c.specs["b2a.2d"] = machineSpec{2, 32, true}
	c.specs["b2a.4d"] = machineSpec{4, 64, true}
	c.specs["b2a.8d"] = machineSpec{8, 128, true}
	c.specs["b2a.16d"] = machineSpec{16, 256, true}
	c.specs["b2a.30d"] = machineSpec{30, 512, true}
	c.specs["b2a.60d"] = machineSpec{60, 1024, true}
	c.specs["b2a.120d"] = machineSpec{120, 2048, true}

	// --- Intel Gen2, Dedicated (d-Suffix) ---
	c.specs["b2i.1d"] = machineSpec{1, 16, true}
	c.specs["b2i.2d"] = machineSpec{2, 32, true}
	c.specs["b2i.4d"] = machineSpec{4, 64, true}
	c.specs["b2i.8d"] = machineSpec{8, 120, true}
	c.specs["b2i.16d"] = machineSpec{16, 238, true}
	c.specs["b2i.30d"] = machineSpec{30, 476, true}
	c.specs["b2i.36d"] = machineSpec{36, 952, true}
}
