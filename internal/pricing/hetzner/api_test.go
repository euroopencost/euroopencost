package hetzner

import (
	"math"
	"testing"

	"github.com/euroopencost/euroopencost/internal/models"
)

func TestParseNetPrice(t *testing.T) {
	cases := []struct {
		input string
		want  float64
	}{
		{"0.0057000000", 0.0057},
		{"0.0050000000", 0.005},
		{"1.190000", 1.19},
		{"", 0},
		{"  ", 0},
		{"not-a-float", 0},
	}
	for _, tc := range cases {
		got := parseNetPrice(tc.input)
		if got != tc.want {
			t.Errorf("input=%q: got %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestGetPriceForResource_KnownServer(t *testing.T) {
	c := NewClient() // HCLOUD_TOKEN not set → uses static prices
	r := models.Resource{
		ServiceName: "hetzner-server",
		APIFlavor:   "cx22",
	}
	price, err := c.GetPriceForResource(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 0.0050 {
		t.Errorf("cx22 price: got %v, want 0.005", price)
	}
}

func TestGetPriceForResource_UnknownServer(t *testing.T) {
	c := NewClient()
	r := models.Resource{
		ServiceName: "hetzner-server",
		APIFlavor:   "nonexistent-type",
	}
	price, err := c.GetPriceForResource(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 0 {
		t.Errorf("unknown server type: got %v, want 0", price)
	}
}

func TestGetPriceForResource_Volume(t *testing.T) {
	c := NewClient()
	r := models.Resource{
		ServiceName: "hetzner-volume",
		Quantity:    100,
	}
	price, err := c.GetPriceForResource(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := (0.0476 * 100) / (24 * 30)
	if math.Abs(price-expected) > 1e-12 {
		t.Errorf("volume price: got %v, want ~%v", price, expected)
	}
}

func TestGetPriceForResource_FreeResource(t *testing.T) {
	c := NewClient()
	r := models.Resource{ServiceName: "hetzner-free", Flavor: "Firewall"}
	price, err := c.GetPriceForResource(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if price != 0 {
		t.Errorf("free resource: got %v, want 0", price)
	}
}
