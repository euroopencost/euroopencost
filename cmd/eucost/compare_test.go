package main

import (
	"testing"

	"github.com/euroopencost/euroopencost/internal/models"
)

func TestDetectProvider_Single(t *testing.T) {
	resources := []models.Resource{
		{Provider: "hetzner"},
		{Provider: "hetzner"},
		{Provider: "hetzner"},
	}
	got := detectProvider(resources)
	if got != "hetzner" {
		t.Errorf("got %q, want %q", got, "hetzner")
	}
}

func TestDetectProvider_Dominant(t *testing.T) {
	resources := []models.Resource{
		{Provider: "otc"},
		{Provider: "otc"},
		{Provider: "hetzner"},
	}
	got := detectProvider(resources)
	if got != "otc" {
		t.Errorf("got %q, want %q", got, "otc")
	}
}

func TestDetectProvider_Empty(t *testing.T) {
	got := detectProvider(nil)
	if got != "unbekannt" {
		t.Errorf("got %q, want %q", got, "unbekannt")
	}
}

func TestDetectProvider_EmptyList(t *testing.T) {
	got := detectProvider([]models.Resource{})
	if got != "unbekannt" {
		t.Errorf("got %q, want %q", got, "unbekannt")
	}
}

func TestDetectProvider_EmptyProviderSkipped(t *testing.T) {
	// Resources with empty Provider should be skipped in counts
	resources := []models.Resource{
		{Provider: ""},
		{Provider: "stackit"},
	}
	got := detectProvider(resources)
	if got != "stackit" {
		t.Errorf("got %q, want %q", got, "stackit")
	}
}
