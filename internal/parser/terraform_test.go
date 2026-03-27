package parser

import (
	"strings"
	"testing"
)

func TestVolumeTypeToAPIFlavor(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"SSD", "vss.ssd"},
		{"SAS", "vss.sas"},
		{"HIGH", "vss.sas"},
		{"GPSSD", "vss.gpssd"},
		{"ESSD", "vss.essd"},
		{"", "vss.ssd"},       // default
		{"UNKNOWN", "vss.ssd"}, // default
	}
	for _, tc := range cases {
		got := volumeTypeToAPIFlavor(tc.input)
		if got != tc.want {
			t.Errorf("input=%q: got %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestExtractEVSInfo_ExplicitType(t *testing.T) {
	p := NewParser()
	values := map[string]interface{}{
		"volume_type": "GPSSD",
		"size":        float64(100),
	}
	display, api, qty := p.extractEVSInfo(values)
	if display != "GPSSD 100GB" {
		t.Errorf("display: got %q, want %q", display, "GPSSD 100GB")
	}
	if api != "vss.gpssd" {
		t.Errorf("api: got %q, want %q", api, "vss.gpssd")
	}
	if qty != 100 {
		t.Errorf("qty: got %v, want %v", qty, 100)
	}
}

func TestExtractEVSInfo_VolumeSizeKey(t *testing.T) {
	p := NewParser()
	values := map[string]interface{}{
		"volume_type": "SSD",
		"volume_size": float64(40),
	}
	display, _, qty := p.extractEVSInfo(values)
	if display != "SSD 40GB" {
		t.Errorf("display: got %q, want %q", display, "SSD 40GB")
	}
	if qty != 40 {
		t.Errorf("qty: got %v, want %v", qty, 40)
	}
}

func TestExtractEVSInfo_DefaultsToSSD(t *testing.T) {
	p := NewParser()
	values := map[string]interface{}{"size": float64(50)}
	_, api, _ := p.extractEVSInfo(values)
	if api != "vss.ssd" {
		t.Errorf("api: got %q, want %q", api, "vss.ssd")
	}
}

func TestParseData_HetznerServer(t *testing.T) {
	p := NewParser()
	planJSON := `{
		"planned_values": {
			"root_module": {
				"resources": [{
					"name": "web",
					"type": "hcloud_server",
					"values": {"server_type": "cx22"}
				}]
			}
		}
	}`
	resources, err := p.parseData([]byte(planJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("got %d resources, want 1", len(resources))
	}
	r := resources[0]
	if r.Provider != "hetzner" {
		t.Errorf("provider: got %q, want %q", r.Provider, "hetzner")
	}
	if r.APIFlavor != "cx22" {
		t.Errorf("APIFlavor: got %q, want %q", r.APIFlavor, "cx22")
	}
}

func TestParseData_OTCWithBootVolume(t *testing.T) {
	p := NewParser()
	planJSON := `{
		"planned_values": {
			"root_module": {
				"resources": [{
					"name": "app",
					"type": "opentelekomcloud_compute_instance_v2",
					"values": {
						"flavor_name": "s3.medium.4",
						"block_device": [{"volume_type": "SSD", "volume_size": 40}]
					}
				}]
			}
		}
	}`
	resources, err := p.parseData([]byte(planJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 2 {
		t.Fatalf("got %d resources, want 2", len(resources))
	}
	hasBootVol := false
	for _, r := range resources {
		if strings.Contains(r.Name, "boot volume") {
			hasBootVol = true
		}
	}
	if !hasBootVol {
		t.Error("expected boot volume resource not found")
	}
}

func TestParseData_UnknownTypeSkipped(t *testing.T) {
	p := NewParser()
	planJSON := `{
		"planned_values": {
			"root_module": {
				"resources": [{
					"name": "x",
					"type": "some_unknown_resource_v99",
					"values": {}
				}]
			}
		}
	}`
	resources, err := p.parseData([]byte(planJSON))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resources) != 0 {
		t.Errorf("got %d resources, want 0", len(resources))
	}
}

func TestParseData_InvalidJSON(t *testing.T) {
	p := NewParser()
	_, err := p.parseData([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}
