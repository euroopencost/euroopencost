package scoring

import (
	"github.com/euroopencost/euroopencost/internal/models"
)

// SovereignInfo enthält Details zum Souveränitäts-Score.
type SovereignInfo struct {
	Score       int     // 0-100%
	EUResources float64 // Summe der Kosten auf EU-Infrastruktur
	USResources float64 // Summe der Kosten auf US-Infrastruktur (Potenzial für Refactoring)
	TotalCost   float64 // Gesamtkosten
	Compliance  float64 // Durchschnittliche Compliance-Gewichtung
}

// ProviderWeights definiert die Compliance- und Risikogewichtung pro Provider.
// S = (sum(R_eu * W_c) - sum(R_us * W_r)) / C_total
var providerWeights = map[string]struct {
	Compliance float64 // W_c (0.0 - 1.0)
	Risk       float64 // W_r (0.0 - 1.0)
	IsEU       bool
}{
	"otc":     {Compliance: 1.0, Risk: 0.05, IsEU: true},  // Telekom: Höchste Compliance
	"hetzner": {Compliance: 0.98, Risk: 0.02, IsEU: true}, // Hetzner: Sehr gut
	"stackit": {Compliance: 1.0, Risk: 0.05, IsEU: true},  // Schwarz IT: Höchste Compliance
	"ionos":   {Compliance: 1.0, Risk: 0.05, IsEU: true},  // IONOS: Höchste Compliance
	"aws":     {Compliance: 0.4, Risk: 0.8, IsEU: false},  // US-Hyperscaler
	"azure":   {Compliance: 0.4, Risk: 0.8, IsEU: false},
	"gcp":     {Compliance: 0.3, Risk: 0.9, IsEU: false},
}

// CalculateSovereignScore berechnet den Score basierend auf der Formel im Visionboard.
func CalculateSovereignScore(resources []models.Resource, total models.Total) SovereignInfo {
	if total.MonthlyPrice == 0 {
		return SovereignInfo{Score: 100}
	}

	var euWeightedSum float64
	var usWeightedSum float64
	var euCost float64
	var usCost float64

	for _, res := range resources {
		weight, ok := providerWeights[res.Provider]
		if !ok {
			// Unbekannte Provider neutral/riskant behandeln
			weight = struct {
				Compliance float64
				Risk       float64
				IsEU       bool
			}{Compliance: 0.5, Risk: 0.5, IsEU: false}
		}

		monthly := res.MonthlyPrice()
		if weight.IsEU {
			euWeightedSum += monthly * weight.Compliance
			euCost += monthly
		} else {
			usWeightedSum += monthly * weight.Risk
			usCost += monthly
		}
	}

	// Formel: S = (sum(R_eu * W_c) - sum(R_us * W_r)) / C_total
	scoreRaw := (euWeightedSum - usWeightedSum) / total.MonthlyPrice
	score := int(scoreRaw * 100)

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return SovereignInfo{
		Score:       score,
		EUResources: euCost,
		USResources: usCost,
		TotalCost:   total.MonthlyPrice,
		Compliance:  scoreRaw, // Vereinfacht
	}
}
