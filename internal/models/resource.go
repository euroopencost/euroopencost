package models

import "strings"

// Resource repräsentiert eine Cloud-Ressource mit Preisinformationen.
type Resource struct {
	Provider    string  // Cloud-Anbieter: "otc", "hetzner", etc.
	Name        string
	Type        string  // z.B. "opentelekomcloud_compute_instance_v2" oder "hcloud_server"
	ServiceName string  // z.B. "ecs", "hetzner-server"
	Flavor      string  // Anzeige-Flavor: z.B. "s3.medium.4" oder "cx21"
	APIFlavor   string  // API-Flavor für Preisabfrage
	Quantity    float64 // Menge für mengenbasierte Preise (z.B. GB), 0 = 1
	HourlyPrice float64 // EUR pro Stunde
}

// MonthlyPrice berechnet den monatlichen Preis (24 Stunden * 30 Tage).
func (r Resource) MonthlyPrice() float64 {
	return r.HourlyPrice * 24 * 30
}

// DisplayType gibt den formatierten Typ für die Ausgabe zurück.
// Kostenpflichtige Ressourcen: "ECS s3.medium.4", "Server cx21"
// Kostenlose Ressourcen: "VPC", "Firewall", etc.
func (r Resource) DisplayType() string {
	serviceDisplayNames := map[string]string{
		// OTC
		"ecs":           "ECS",
		"evs":           "EVS",
		"eip":           "EIP",
		"elb":           "ELB",
		"rds":           "RDS",
		"nat":           "NAT",
		"dcs":           "DCS",
		"obs":           "OBS",
		"cce":           "CCE",
		"vpc":           "",
		"vpc-subnet":    "",
		"secgroup":      "",
		"secgroup-rule": "",
		// Hetzner
		"hetzner-server":     "Server",
		"hetzner-volume":     "Volume",
		"hetzner-floatingip": "Floating IP",
		"hetzner-lb":         "Load Balancer",
		"hetzner-free":       "",
		// STACKIT
		"stackit-server": "Server",
		"stackit-volume": "Volume",
		"stackit-obs":    "Object Storage",
		"stackit-free":   "",
		// IONOS Cloud
		"ionos-server": "Server",
		"ionos-volume": "Volume",
		"ionos-ip":     "IP Block",
		"ionos-free":   "",
	}

	displayName, known := serviceDisplayNames[r.ServiceName]
	if !known {
		return r.Flavor
	}

	if displayName == "" {
		// Kostenlose Ressourcen → nur Flavor
		return r.Flavor
	}

	if r.Flavor == "" {
		return displayName
	}
	return strings.TrimSpace(displayName + " " + r.Flavor)
}

// Total enthält die Gesamtkosten aller Ressourcen.
type Total struct {
	HourlyPrice  float64
	MonthlyPrice float64
}
