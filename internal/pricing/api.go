package pricing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// PriceEntry repräsentiert einen einzelnen Preiseintrag aus der OTC Pricing API.
type PriceEntry struct {
	OpiFlavour  string `json:"opiFlavour"`
	PriceAmount string `json:"priceAmount"`
	Unit        string `json:"unit"`
}

// apiResponse ist die Top-Level-Struktur der OTC API-Antwort.
type apiResponse struct {
	Response struct {
		Result map[string][]PriceEntry `json:"result"`
	} `json:"response"`
}

// Client ist der OTC Pricing API Client mit In-Memory Cache.
type Client struct {
	cache      map[string][]PriceEntry
	httpClient *http.Client
}

// NewClient erstellt einen neuen API-Client mit 10 Sekunden Timeout.
func NewClient() *Client {
	return &Client{
		cache: make(map[string][]PriceEntry),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetPrices ruft die Preise für einen OTC-Service ab.
// Verwendet In-Memory Cache — API wird pro Service nur einmal aufgerufen.
// Gibt nil zurück wenn der Service kostenlos ist (HTTP 500 von der API).
func (c *Client) GetPrices(serviceName string) ([]PriceEntry, error) {
	// Cache prüfen
	if cached, ok := c.cache[serviceName]; ok {
		return cached, nil
	}

	// URL aufbauen
	params := url.Values{}
	params.Set("responseFormat", "json")
	params.Set("serviceName[0]", serviceName)
	params.Set("region[1]", "eu-de")
	fullURL := "https://calculator.otc-service.com/en/open-telekom-price-api/?" + params.Encode()

	resp, err := c.httpClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("API-Anfrage fehlgeschlagen für '%s': %w", serviceName, err)
	}
	defer resp.Body.Close()

	// HTTP 500 bedeutet kostenloser Service (kein Preiseintrag vorhanden)
	if resp.StatusCode == http.StatusInternalServerError {
		c.cache[serviceName] = nil
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API-Fehler für '%s': HTTP %d", serviceName, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("Antwort lesen fehlgeschlagen: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("JSON parsen fehlgeschlagen: %w", err)
	}

	entries := apiResp.Response.Result[serviceName]
	c.cache[serviceName] = entries
	return entries, nil
}

// ParsePrice parst einen Preisstring aus der API (z.B. "0.052900 EUR") in einen float64.
func ParsePrice(priceAmount string) (float64, error) {
	fields := strings.Fields(priceAmount)
	if len(fields) == 0 {
		return 0, fmt.Errorf("leerer Preisstring")
	}
	price, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("Preis parsen fehlgeschlagen '%s': %w", fields[0], err)
	}
	return price, nil
}
