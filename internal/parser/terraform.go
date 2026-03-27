package parser

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/euroopencost/euroopencost/internal/models"
)

// TerraformPlan repräsentiert die Top-Level-Struktur eines Terraform Plan JSON.
type TerraformPlan struct {
	PlannedValues PlannedValues `json:"planned_values"`
}

// PlannedValues enthält das Root-Modul des Plans.
type PlannedValues struct {
	RootModule RootModule `json:"root_module"`
}

// RootModule enthält alle geplanten Ressourcen.
type RootModule struct {
	Resources []TerraformResource `json:"resources"`
}

// TerraformResource repräsentiert eine einzelne Terraform-Ressource im Plan.
type TerraformResource struct {
	Name   string                 `json:"name"`
	Type   string                 `json:"type"`
	Values map[string]interface{} `json:"values"`
}

// Parser liest Terraform Plan JSON-Dateien ein.
type Parser struct{}

// NewParser erstellt einen neuen Parser.
func NewParser() *Parser {
	return &Parser{}
}

// Parse liest eine Terraform Plan JSON-Datei ein und gibt alle erkannten Ressourcen zurück.
func (p *Parser) Parse(filename string) ([]models.Resource, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Datei lesen fehlgeschlagen: %w", err)
	}
	return p.parseData(data)
}

// ParseReader liest einen Terraform Plan JSON aus einem io.Reader (z.B. os.Stdin).
func (p *Parser) ParseReader(r io.Reader) ([]models.Resource, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("Eingabe lesen fehlgeschlagen: %w", err)
	}
	return p.parseData(data)
}

// parseData verarbeitet Terraform Plan JSON-Bytes und gibt alle erkannten Ressourcen zurück.
func (p *Parser) parseData(data []byte) ([]models.Resource, error) {
	var plan TerraformPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("JSON parsen fehlgeschlagen: %w", err)
	}

	var resources []models.Resource

	for _, tfRes := range plan.PlannedValues.RootModule.Resources {
		res, err := p.parseResource(tfRes)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warnung: Ressource '%s' (%s) übersprungen: %v\n", tfRes.Name, tfRes.Type, err)
			continue
		}
		if res == nil {
			// Unbekannter Typ — bereits gewarnt in parseResource
			continue
		}

		resources = append(resources, *res)

		// Boot Volume automatisch aus ECS-Instanz extrahieren
		if tfRes.Type == "opentelekomcloud_compute_instance_v2" {
			if bootVol := p.extractBootVolume(tfRes); bootVol != nil {
				resources = append(resources, *bootVol)
			}
		}
	}

	return resources, nil
}

// parseResource mappt eine Terraform-Ressource auf ein internes Resource-Modell.
func (p *Parser) parseResource(tfRes TerraformResource) (*models.Resource, error) {
	res := &models.Resource{
		Type:     tfRes.Type,
		Provider: "otc", // Standard: OTC; wird für Hetzner überschrieben
	}

	switch tfRes.Type {
	case "opentelekomcloud_compute_instance_v2":
		res.Name = tfRes.Name
		res.ServiceName = "ecs"
		// flavor_name bevorzugen, sonst flavor_id
		if flavorName, ok := tfRes.Values["flavor_name"].(string); ok && flavorName != "" {
			res.Flavor = flavorName
			res.APIFlavor = flavorName
		} else if flavorID, ok := tfRes.Values["flavor_id"].(string); ok && flavorID != "" {
			res.Flavor = flavorID
			res.APIFlavor = flavorID
		}

	case "opentelekomcloud_blockstorage_volume_v2":
		res.Name = tfRes.Name
		res.ServiceName = "evs"
		res.Flavor, res.APIFlavor, res.Quantity = p.extractEVSInfo(tfRes.Values)

	case "opentelekomcloud_vpc_eip_v1":
		res.Name = tfRes.Name
		res.ServiceName = "eip"
		// EIP hat einen festen Flavor "eip" in der API (pro Stunde)
		var bwSize float64
		if bwList, ok := tfRes.Values["bandwidth"].([]interface{}); ok && len(bwList) > 0 {
			if bw, ok := bwList[0].(map[string]interface{}); ok {
				bwSize, _ = bw["size"].(float64)
			}
		}
		if bwSize > 0 {
			res.Flavor = fmt.Sprintf("%.0fMbps", bwSize)
		} else {
			res.Flavor = "EIP"
		}
		res.APIFlavor = "eip"

	case "opentelekomcloud_lb_loadbalancer_v2":
		res.Name = tfRes.Name
		res.ServiceName = "elb"
		res.Flavor = "ELB"

	case "opentelekomcloud_rds_instance_v3":
		res.Name = tfRes.Name
		res.ServiceName = "rds"
		flavor, _ := tfRes.Values["flavor"].(string)
		res.Flavor = flavor

	case "opentelekomcloud_nat_gateway_v2":
		res.Name = tfRes.Name
		res.ServiceName = "nat"
		spec, _ := tfRes.Values["spec"].(string)
		res.Flavor = spec

	case "opentelekomcloud_dcs_instance_v1":
		res.Name = tfRes.Name
		res.ServiceName = "dcs"
		engine, _ := tfRes.Values["engine"].(string)
		capacity, _ := tfRes.Values["capacity"].(float64)
		res.Flavor = fmt.Sprintf("%s %.0fGB", engine, capacity)

	case "opentelekomcloud_obs_bucket":
		res.Name = tfRes.Name
		res.ServiceName = "obs"
		res.Flavor = "OBS"

	case "opentelekomcloud_cce_cluster_v3":
		res.Name = tfRes.Name
		res.ServiceName = "cce"
		flavorID, _ := tfRes.Values["flavor_id"].(string)
		res.Flavor = flavorID

	// Kostenlose Ressourcen
	case "opentelekomcloud_vpc_v1":
		res.Name = tfRes.Name + " (vpc)"
		res.ServiceName = "vpc"
		res.Flavor = "VPC"

	case "opentelekomcloud_vpc_subnet_v1":
		res.Name = tfRes.Name + " (subnet)"
		res.ServiceName = "vpc-subnet"
		res.Flavor = "VPC Subnet"

	case "opentelekomcloud_networking_secgroup_v2":
		res.Name = tfRes.Name + " (security group)"
		res.ServiceName = "secgroup"
		res.Flavor = "Security Group"

	case "opentelekomcloud_networking_secgroup_rule_v2":
		res.Name = tfRes.Name + " (security group rule)"
		res.ServiceName = "secgroup-rule"
		res.Flavor = "Security Group Rule"

	// --- Hetzner Cloud ---
	case "hcloud_server":
		res.Provider = "hetzner"
		res.Name = tfRes.Name
		res.ServiceName = "hetzner-server"
		serverType, _ := tfRes.Values["server_type"].(string)
		res.Flavor = serverType
		res.APIFlavor = serverType

	case "hcloud_volume":
		res.Provider = "hetzner"
		res.Name = tfRes.Name
		res.ServiceName = "hetzner-volume"
		size, _ := tfRes.Values["size"].(float64)
		res.Flavor = fmt.Sprintf("%.0fGB", size)
		res.Quantity = size

	case "hcloud_floating_ip":
		res.Provider = "hetzner"
		res.Name = tfRes.Name
		res.ServiceName = "hetzner-floatingip"
		ipType, _ := tfRes.Values["type"].(string)
		if ipType == "" {
			ipType = "ipv4"
		}
		res.Flavor = strings.ToUpper(ipType)

	case "hcloud_load_balancer":
		res.Provider = "hetzner"
		res.Name = tfRes.Name
		res.ServiceName = "hetzner-lb"
		lbType, _ := tfRes.Values["load_balancer_type"].(string)
		res.Flavor = lbType
		res.APIFlavor = lbType

	// --- AWS (For Policy Testing) ---
	case "aws_instance":
		res.Provider = "aws"
		res.Name = tfRes.Name
		res.ServiceName = "aws-ec2"
		res.Flavor, _ = tfRes.Values["instance_type"].(string)

	// Kostenlose Hetzner Ressourcen
	case "hcloud_firewall":
		res.Provider = "hetzner"
		res.Name = tfRes.Name + " (firewall)"
		res.ServiceName = "hetzner-free"
		res.Flavor = "Firewall"

	case "hcloud_network", "hcloud_network_subnet":
		res.Provider = "hetzner"
		res.Name = tfRes.Name + " (network)"
		res.ServiceName = "hetzner-free"
		res.Flavor = "Network"

	case "hcloud_ssh_key":
		res.Provider = "hetzner"
		res.Name = tfRes.Name + " (ssh key)"
		res.ServiceName = "hetzner-free"
		res.Flavor = "SSH Key"

	// --- STACKIT ---
	case "stackit_server":
		res.Provider = "stackit"
		res.Name = tfRes.Name
		res.ServiceName = "stackit-server"
		machineType, _ := tfRes.Values["machine_type"].(string)
		res.Flavor = machineType
		res.APIFlavor = machineType

	case "stackit_volume":
		res.Provider = "stackit"
		res.Name = tfRes.Name
		res.ServiceName = "stackit-volume"
		size, _ := tfRes.Values["size"].(float64)
		res.Flavor = fmt.Sprintf("%.0fGB", size)
		res.Quantity = size

	case "stackit_object_storage_bucket":
		res.Provider = "stackit"
		res.Name = tfRes.Name
		res.ServiceName = "stackit-obs"

	// Kostenlose STACKIT Ressourcen
	case "stackit_security_group":
		res.Provider = "stackit"
		res.Name = tfRes.Name + " (security group)"
		res.ServiceName = "stackit-free"
		res.Flavor = "Security Group"

	case "stackit_security_group_rule":
		res.Provider = "stackit"
		res.Name = tfRes.Name + " (security group rule)"
		res.ServiceName = "stackit-free"
		res.Flavor = "Security Group Rule"

	case "stackit_server_volume_attach":
		res.Provider = "stackit"
		res.Name = tfRes.Name + " (volume attach)"
		res.ServiceName = "stackit-free"
		res.Flavor = "Volume Attach"

	case "stackit_network", "stackit_network_interface":
		res.Provider = "stackit"
		res.Name = tfRes.Name + " (network)"
		res.ServiceName = "stackit-free"
		res.Flavor = "Network"

	// --- IONOS Cloud ---
	case "ionoscloud_server":
		res.Provider = "ionos"
		res.Name = tfRes.Name
		res.ServiceName = "ionos-server"
		cores, _ := tfRes.Values["cores"].(float64)
		ramMB, _ := tfRes.Values["ram"].(float64)
		res.Flavor = fmt.Sprintf("%.0fvCPU %.0fMB RAM", cores, ramMB)
		res.APIFlavor = fmt.Sprintf("cores:%.0f,ram:%.0f", cores, ramMB)

	case "ionoscloud_volume":
		res.Provider = "ionos"
		res.Name = tfRes.Name
		res.ServiceName = "ionos-volume"
		size, _ := tfRes.Values["size"].(float64)
		diskType, _ := tfRes.Values["disk_type"].(string)
		if diskType == "" {
			diskType = "HDD"
		}
		res.Flavor = fmt.Sprintf("%.0fGB %s", size, diskType)
		res.APIFlavor = diskType
		res.Quantity = size

	case "ionoscloud_ipblock":
		res.Provider = "ionos"
		res.Name = tfRes.Name
		res.ServiceName = "ionos-ip"
		ipCount, _ := tfRes.Values["size"].(float64)
		res.Flavor = fmt.Sprintf("%.0f IPs", ipCount)
		res.Quantity = ipCount

	// Kostenlose IONOS Ressourcen
	case "ionoscloud_datacenter":
		res.Provider = "ionos"
		res.Name = tfRes.Name + " (datacenter)"
		res.ServiceName = "ionos-free"
		res.Flavor = "Datacenter"

	case "ionoscloud_lan":
		res.Provider = "ionos"
		res.Name = tfRes.Name + " (lan)"
		res.ServiceName = "ionos-free"
		res.Flavor = "LAN"

	case "ionoscloud_nic":
		res.Provider = "ionos"
		res.Name = tfRes.Name + " (nic)"
		res.ServiceName = "ionos-free"
		res.Flavor = "NIC"

	default:
		fmt.Fprintf(os.Stderr, "Warnung: Unbekannter Ressource-Typ '%s' (%s) - wird übersprungen\n", tfRes.Type, tfRes.Name)
		return nil, nil
	}

	return res, nil
}

// extractBootVolume extrahiert das Boot-Volume aus einer ECS-Instanz als separate EVS-Ressource.
func (p *Parser) extractBootVolume(tfRes TerraformResource) *models.Resource {
	blockDevices, ok := tfRes.Values["block_device"].([]interface{})
	if !ok || len(blockDevices) == 0 {
		return nil
	}

	bd, ok := blockDevices[0].(map[string]interface{})
	if !ok {
		return nil
	}

	displayFlavor, apiFlavor, quantity := p.extractEVSInfo(bd)

	return &models.Resource{
		Name:        tfRes.Name + " (boot volume)",
		Type:        "opentelekomcloud_blockstorage_volume_v2",
		Provider:    "otc",
		ServiceName: "evs",
		Flavor:      displayFlavor,
		APIFlavor:   apiFlavor,
		Quantity:    quantity,
	}
}

// volumeTypeToAPIFlavor mappt den Terraform volume_type auf den OTC API Flavor.
func volumeTypeToAPIFlavor(volumeType string) string {
	switch volumeType {
	case "SSD":
		return "vss.ssd"
	case "SAS", "HIGH":
		return "vss.sas"
	case "GPSSD":
		return "vss.gpssd"
	case "ESSD":
		return "vss.essd"
	default:
		return "vss.ssd" // SSD als Standardwert
	}
}

// extractEVSInfo gibt display Flavor, API Flavor und Menge (GB) für ein EVS-Volume zurück.
func (p *Parser) extractEVSInfo(values map[string]interface{}) (string, string, float64) {
	volumeType, _ := values["volume_type"].(string)
	if volumeType == "" {
		volumeType = "SSD"
	}

	var size float64
	// block_device nutzt "volume_size", blockstorage_volume_v2 nutzt "size"
	if s, ok := values["volume_size"].(float64); ok {
		size = s
	} else if s, ok := values["size"].(float64); ok {
		size = s
	}

	apiFlavor := volumeTypeToAPIFlavor(volumeType)

	var displayFlavor string
	if size > 0 {
		displayFlavor = fmt.Sprintf("%s %.0fGB", volumeType, size)
	} else {
		displayFlavor = volumeType
	}

	return displayFlavor, apiFlavor, size
}
