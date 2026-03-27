package ionos

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/euroopencost/euroopencost/internal/models"
)

// pricingResponse ist die API-Antwort für GET /cloudapi/v6/pricing.
type pricingResponse struct {
	Items []pricingItem `json:"items"`
}

type pricingItem struct {
	ResourceHRef string          `json:"resourceHref"`
	ResourceType string          `json:"resourceType"`
	Price        []resourcePrice `json:"price"`
}

type resourcePrice struct {
	Location    string `json:"location"`
	PricePerUnit string `json:"pricePerUnit"`
	Unit        string `json:"unit"`
}

// Client ist der IONOS Cloud Pricing Client.
// Mit IONOS_TOKEN werden Live-Preise von der API geladen,
// ohne Token werden eingebettete Fallback-Preise verwendet.
type Client struct {
	httpClient    *http.Client
	token         string
	ratePerCore   float64 // EUR/h pro vCPU
	ratePerGBRAM  float64 // EUR/h pro GB RAM
	volumeHDDPrice float64 // EUR/GB/Monat (HDD)
	volumeSSDPrice float64 // EUR/GB/Monat (SSD Standard)
	volumeSSDPremiumPrice float64 // EUR/GB/Monat (SSD Premium)
	ratePerIP     float64 // EUR/h pro IP-Adresse
	loaded        bool
}

// NewClient erstellt einen neuen IONOS Client.
// Token wird aus IONOS_TOKEN Umgebungsvariable gelesen (Format: "user:password" oder "token").
func NewClient() *Client {
	return &Client{
		token: os.Getenv("IONOS_TOKEN"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetPriceForResource gibt den stündlichen Preis für eine IONOS-Ressource zurück.
func (c *Client) GetPriceForResource(res models.Resource) (float64, error) {
	if err := c.loadPrices(); err != nil {
		return 0, err
	}

	switch res.ServiceName {
	case "ionos-server":
		// APIFlavor ist kodiert als "cores:4,ram:8192"
		cores, ramMB, err := decodeServerFlavor(res.APIFlavor)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warnung: IONOS Server-Flavor konnte nicht dekodiert werden '%s': %v\n", res.APIFlavor, err)
			return 0, nil
		}
		ramGB := ramMB / 1024.0
		return cores*c.ratePerCore + ramGB*c.ratePerGBRAM, nil

	case "ionos-volume":
		// Preis: EUR/GB/Monat → EUR/h (720 Stunden/Monat)
		if res.Quantity > 0 {
			var monthlyRate float64
			switch strings.ToUpper(res.APIFlavor) {
			case "SSD", "SSD_STANDARD", "SSD STANDARD":
				monthlyRate = c.volumeSSDPrice
			case "SSD_PREMIUM", "SSD PREMIUM":
				monthlyRate = c.volumeSSDPremiumPrice
			default: // "HDD" oder unbekannt
				monthlyRate = c.volumeHDDPrice
			}
			return (monthlyRate * res.Quantity) / 720, nil
		}
		return 0, nil

	case "ionos-ip":
		// IP-Block: Quantity = Anzahl IPs
		if res.Quantity > 0 {
			return c.ratePerIP * res.Quantity, nil
		}
		return c.ratePerIP, nil

	case "ionos-free":
		return 0, nil
	}

	return 0, nil
}

// loadPrices lädt IONOS-Preise einmalig (API oder statische Fallback-Preise).
func (c *Client) loadPrices() error {
	if c.loaded {
		return nil
	}

	if c.token == "" {
		fmt.Fprintln(os.Stderr, "Info: IONOS_TOKEN nicht gesetzt - verwende eingebettete IONOS-Preise (Stand 2025, netto)")
		c.loadStaticPrices()
		c.loaded = true
		return nil
	}

	if err := c.loadFromAPI(); err != nil {
		fmt.Fprintf(os.Stderr, "Warnung: IONOS Pricing API fehlgeschlagen (%v) - verwende eingebettete Preise\n", err)
		c.loadStaticPrices()
	}

	c.loaded = true
	return nil
}

// loadFromAPI lädt Preise von der IONOS Cloud API.
func (c *Client) loadFromAPI() error {
	data, err := c.doRequest("https://api.ionos.com/cloudapi/v6/pricing")
	if err != nil {
		return err
	}

	var resp pricingResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return fmt.Errorf("IONOS Pricing API JSON parsen fehlgeschlagen: %w", err)
	}

	// Preise aus API-Antwort extrahieren
	for _, item := range resp.Items {
		if len(item.Price) == 0 {
			continue
		}
		price := parsePrice(item.Price[0].PricePerUnit)

		switch item.ResourceType {
		case "CORE":
			c.ratePerCore = price
		case "RAM":
			// API gibt EUR/MB/h — umrechnen auf EUR/GB/h
			c.ratePerGBRAM = price * 1024
		case "HDD":
			c.volumeHDDPrice = price * 720 // EUR/h → EUR/Monat
		case "SSD":
			c.volumeSSDPrice = price * 720
		case "SSD_PREMIUM":
			c.volumeSSDPremiumPrice = price * 720
		case "IP":
			c.ratePerIP = price
		}
	}

	// Fehlende Werte mit Fallback auffüllen
	if c.ratePerCore == 0 {
		c.loadStaticPrices()
	}

	return nil
}

// loadStaticPrices lädt eingebettete IONOS-Preise.
// Quellen: https://cloud.ionos.com/prices — Netto-Preise, Stand 2025.
// TODO: Preise bei neuer IONOS-Preisliste aktualisieren.
func (c *Client) loadStaticPrices() {
	c.ratePerCore = 0.0059         // EUR/h pro vCPU
	c.ratePerGBRAM = 0.0029        // EUR/h pro GB RAM
	c.volumeHDDPrice = 0.0330      // EUR/GB/Monat
	c.volumeSSDPrice = 0.0680      // EUR/GB/Monat (SSD Standard)
	c.volumeSSDPremiumPrice = 0.0990 // EUR/GB/Monat (SSD Premium)
	c.ratePerIP = 0.0040           // EUR/h pro IP-Adresse
}

// doRequest führt einen authentifizierten API-Request an IONOS aus.
func (c *Client) doRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// IONOS nutzt Basic Auth mit "username:password" oder "token" als Passwort
	req.Header.Set("Authorization", "Basic "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d von IONOS API", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// decodeServerFlavor dekodiert den kodierten APIFlavor "cores:4,ram:8192".
func decodeServerFlavor(flavor string) (cores float64, ramMB float64, err error) {
	parts := strings.Split(flavor, ",")
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		val, parseErr := strconv.ParseFloat(strings.TrimSpace(kv[1]), 64)
		if parseErr != nil {
			return 0, 0, fmt.Errorf("ungültiger Wert '%s': %w", kv[1], parseErr)
		}
		switch strings.TrimSpace(kv[0]) {
		case "cores":
			cores = val
		case "ram":
			ramMB = val
		}
	}
	if cores == 0 {
		return 0, 0, fmt.Errorf("kein 'cores' Wert in Flavor '%s'", flavor)
	}
	return cores, ramMB, nil
}

// parsePrice parst einen Preis-String in float64.
func parsePrice(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	price, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return price
}
