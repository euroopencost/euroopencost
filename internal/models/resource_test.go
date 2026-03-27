package models

import (
	"strings"
	"testing"
)

func TestMonthlyPrice(t *testing.T) {
	cases := []struct {
		hourly float64
		want   float64
	}{
		{0.0, 0.0},
		{1.0, 720.0},
		{0.0057, 0.0057 * 24 * 30},
		{0.052900, 0.052900 * 24 * 30},
	}
	for _, tc := range cases {
		r := Resource{HourlyPrice: tc.hourly}
		got := r.MonthlyPrice()
		if got != tc.want {
			t.Errorf("HourlyPrice=%v: got %v, want %v", tc.hourly, got, tc.want)
		}
	}
}

func TestDisplayType_KnownServices(t *testing.T) {
	cases := []struct {
		serviceName string
		flavor      string
		want        string
	}{
		// OTC — displayName + flavor
		{"ecs", "s3.medium.4", "ECS s3.medium.4"},
		{"evs", "SSD 40GB", "EVS SSD 40GB"},
		{"eip", "EIP", "EIP EIP"},
		{"elb", "Small", "ELB Small"},
		{"rds", "rds.c2.medium", "RDS rds.c2.medium"},
		{"nat", "NAT", "NAT NAT"},
		{"dcs", "Redis", "DCS Redis"},
		{"obs", "Standard", "OBS Standard"},
		{"cce", "Node", "CCE Node"},
		// OTC — empty displayName → returns Flavor
		{"vpc", "VPC", "VPC"},
		{"vpc-subnet", "subnet", "subnet"},
		{"secgroup", "sg-01", "sg-01"},
		{"secgroup-rule", "rule", "rule"},
		// Hetzner — displayName + flavor
		{"hetzner-server", "cx22", "Server cx22"},
		{"hetzner-volume", "10GB", "Volume 10GB"},
		{"hetzner-floatingip", "IPv4", "Floating IP IPv4"},
		{"hetzner-lb", "lb11", "Load Balancer lb11"},
		// Hetzner — empty displayName → returns Flavor
		{"hetzner-free", "Firewall", "Firewall"},
		// STACKIT
		{"stackit-server", "c1.2", "Server c1.2"},
		{"stackit-volume", "50GB", "Volume 50GB"},
		{"stackit-obs", "bucket", "Object Storage bucket"},
		{"stackit-free", "sg", "sg"},
		// IONOS
		{"ionos-server", "4vCPU 8192MB RAM", "Server 4vCPU 8192MB RAM"},
		{"ionos-volume", "100GB", "Volume 100GB"},
		{"ionos-ip", "IPv4", "IP Block IPv4"},
		{"ionos-free", "LAN", "LAN"},
	}
	for _, tc := range cases {
		r := Resource{ServiceName: tc.serviceName, Flavor: tc.flavor}
		got := r.DisplayType()
		if got != tc.want {
			t.Errorf("ServiceName=%q Flavor=%q: got %q, want %q", tc.serviceName, tc.flavor, got, tc.want)
		}
	}
}

func TestDisplayType_UnknownService(t *testing.T) {
	r := Resource{ServiceName: "unknown-xyz", Flavor: "my-flavor"}
	got := r.DisplayType()
	if got != "my-flavor" {
		t.Errorf("got %q, want %q", got, "my-flavor")
	}
}

func TestDisplayType_EmptyDisplayName_EmptyFlavor(t *testing.T) {
	// ServiceName with empty displayName AND empty Flavor → empty string
	r := Resource{ServiceName: "vpc", Flavor: ""}
	got := r.DisplayType()
	if strings.TrimSpace(got) != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

func TestDisplayType_KnownService_EmptyFlavor(t *testing.T) {
	// stackit-obs with no flavor → only displayName
	r := Resource{ServiceName: "stackit-obs", Flavor: ""}
	got := r.DisplayType()
	if got != "Object Storage" {
		t.Errorf("got %q, want %q", got, "Object Storage")
	}
}
