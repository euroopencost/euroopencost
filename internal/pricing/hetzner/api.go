package hetzner

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

// serverTypesResponse ist die API-Antwort für GET /v1/server_types.
type serverTypesResponse struct {
	ServerTypes []serverType `json:"server_types"`
}

type serverType struct {
	Name   string       `json:"name"`
	Prices []priceEntry `json:"prices"`
}

type priceEntry struct {
	Location     string    `json:"location"`
	PriceHourly  priceNet  `json:"price_hourly"`
	PriceMonthly priceNet  `json:"price_monthly"`
}

type priceNet struct {
	Net   string `json:"net"`
	Gross string `json:"gross"`
}

// pricingResponse ist die API-Antwort für GET /v1/pricing.
type pricingResponse struct {
	Pricing struct {
		ServerTypes []struct {
			ID     int          `json:"id"`
			Name   string       `json:"name"`
			Prices []priceEntry `json:"prices"`
		} `json:"server_types"`
		Volume struct {
			PricePerGBMonthly priceNet `json:"price_per_gb_month"`
		} `json:"volume"`
		FloatingIP struct {
			PriceMonthly priceNet `json:"price_monthly"`
		} `json:"floating_ip"`
		LoadBalancerTypes []struct {
			ID     int          `json:"id"`
			Name   string       `json:"name"`
			Prices []priceEntry `json:"prices"`
		} `json:"load_balancer_types"`
	} `json:"pricing"`
}

// Client ist der Hetzner Cloud API Client.
type Client struct {
	token        string
	httpClient   *http.Client
	serverPrices map[string]float64 // serverType → EUR/h
	volumePrice  float64            // EUR/GB/Monat
	floatingIP   float64            // EUR/Monat
	lbPrices     map[string]float64 // lbType → EUR/h
	loaded       bool
}

// NewClient erstellt einen neuen Hetzner API Client.
// Token wird aus HCLOUD_TOKEN Umgebungsvariable gelesen.
func NewClient() *Client {
	return &Client{
		token:        os.Getenv("HCLOUD_TOKEN"),
		serverPrices: make(map[string]float64),
		lbPrices:     make(map[string]float64),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetPriceForResource gibt den stündlichen Preis für eine Hetzner-Ressource zurück.
func (c *Client) GetPriceForResource(res models.Resource) (float64, error) {
	if err := c.loadPrices(); err != nil {
		return 0, err
	}

	switch res.ServiceName {
	case "hetzner-server":
		price, ok := c.serverPrices[res.APIFlavor]
		if !ok {
			fmt.Fprintf(os.Stderr, "Warnung: Kein Hetzner-Preis für Server-Typ '%s'\n", res.APIFlavor)
			return 0, nil
		}
		return price, nil

	case "hetzner-volume":
		// Preis: EUR/GB/Monat → EUR/h
		if res.Quantity > 0 {
			return (c.volumePrice * res.Quantity) / (24 * 30), nil
		}
		return 0, nil

	case "hetzner-floatingip":
		// Preis: EUR/Monat → EUR/h
		return c.floatingIP / (24 * 30), nil

	case "hetzner-lb":
		price, ok := c.lbPrices[res.APIFlavor]
		if !ok {
			fmt.Fprintf(os.Stderr, "Warnung: Kein Hetzner-Preis für LB-Typ '%s'\n", res.APIFlavor)
			return 0, nil
		}
		return price, nil

	case "hetzner-free":
		return 0, nil
	}

	return 0, nil
}

// loadPrices lädt alle Hetzner-Preise einmalig aus der API.
func (c *Client) loadPrices() error {
	if c.loaded {
		return nil
	}

	if c.token == "" {
		fmt.Fprintln(os.Stderr, "Info: HCLOUD_TOKEN nicht gesetzt - verwende eingebettete Hetzner-Preise (Stand 2025, netto)")
		c.loadStaticPrices()
		c.loaded = true
		return nil
	}

	resp, err := c.doRequest("https://api.hetzner.cloud/v1/pricing")
	if err != nil {
		return fmt.Errorf("Hetzner Pricing API fehlgeschlagen: %w", err)
	}

	var pr pricingResponse
	if err := json.Unmarshal(resp, &pr); err != nil {
		return fmt.Errorf("Hetzner Pricing API JSON parsen fehlgeschlagen: %w", err)
	}

	// Server-Preise laden
	for _, st := range pr.Pricing.ServerTypes {
		if len(st.Prices) > 0 {
			price := parseNetPrice(st.Prices[0].PriceHourly.Net)
			c.serverPrices[st.Name] = price
		}
	}

	// Volume-Preis laden (EUR/GB/Monat)
	c.volumePrice = parseNetPrice(pr.Pricing.Volume.PricePerGBMonthly.Net)

	// Floating IP Preis (EUR/Monat)
	c.floatingIP = parseNetPrice(pr.Pricing.FloatingIP.PriceMonthly.Net)

	// Load Balancer Preise
	for _, lb := range pr.Pricing.LoadBalancerTypes {
		if len(lb.Prices) > 0 {
			price := parseNetPrice(lb.Prices[0].PriceHourly.Net)
			c.lbPrices[lb.Name] = price
		}
	}

	c.loaded = true
	return nil
}

// loadStaticPrices lädt eingebettete Hetzner-Preise (Fallback ohne API-Token).
// Quellen: https://www.hetzner.com/cloud/ — Netto-Preise, Stand 2025.
func (c *Client) loadStaticPrices() {
	c.serverPrices = map[string]float64{
		// Shared Intel (CX)
		"cx22": 0.0050, "cx32": 0.0092, "cx42": 0.0185, "cx52": 0.0353,
		// Shared AMD (CPX)
		"cpx11": 0.0050, "cpx21": 0.0092, "cpx31": 0.0168, "cpx41": 0.0294, "cpx51": 0.0555,
		// Shared ARM (CAX)
		"cax11": 0.0042, "cax21": 0.0084, "cax31": 0.0168, "cax41": 0.0319,
		// Dedicated AMD (CCX)
		"ccx13": 0.0168, "ccx23": 0.0319, "ccx33": 0.0605,
		"ccx43": 0.1193, "ccx53": 0.2269, "ccx63": 0.4370,
		// Legacy CX (noch verbreitet in Terraform configs)
		"cx11": 0.0042, "cx21": 0.0076, "cx31": 0.0143, "cx41": 0.0261, "cx51": 0.0471,
	}
	c.volumePrice = 0.0476 // EUR/GB/Monat
	c.floatingIP = 1.190   // EUR/Monat
	c.lbPrices = map[string]float64{
		"lb11": 0.0076, "lb21": 0.0261, "lb31": 0.0471,
	}
}

// doRequest führt einen authentifizierten API-Request aus.
func (c *Client) doRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d von Hetzner API", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// parseNetPrice parst einen Netto-Preis-String (z.B. "0.0057000000") in float64.
func parseNetPrice(s string) float64 {
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
