# EuroOpenCost

**The open-source Terraform cost estimator for European Cloud infrastructure.**

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Providers](https://img.shields.io/badge/Providers-4_EU_Clouds-green)](https://euroopencost.eu)
[![Website](https://img.shields.io/badge/Website-euroopencost.eu-002068)](https://euroopencost.eu)

```
eucost plan plan.json
```

```
+---------------------------+-------------------+------------+-----------+
| RESSOURCE                 | TYP               | EUR/STUNDE | EUR/MONAT |
+---------------------------+-------------------+------------+-----------+
| main                      | ECS s3.medium.4   |     0.0529 |     38.09 |
| main (boot volume)        | EVS SSD 20GB      |     0.0030 |      2.20 |
| main (eip)                | EIP               |     0.0044 |      3.17 |
| cce-cluster               | CCE cce.s2.small  |     0.3800 |    273.60 |
| main (security group)     | Security Group    |     0.0000 |      0.00 |
| main (vpc)                | VPC               |     0.0000 |      0.00 |
+---------------------------+-------------------+------------+-----------+
| TOTAL                     |                   |     0.4403 |    317.06 |
+---------------------------+-------------------+------------+-----------+
```

---

## Why EuroOpenCost?

Most cloud cost tools are built around AWS, Azure, and GCP. European cloud providers —
**OpenTelekomCloud, Hetzner, STACKIT, IONOS** — are either missing entirely or treated
as second-class citizens.

EuroOpenCost is different:

| Problem | How we solve it |
|---------|----------------|
| US-centric tools ignore EU providers | Native support for 4 EU cloud providers |
| Cost estimations need a SaaS account | Zero signup, no cloud credentials required |
| No Terraform-native EU cost visibility | Read `terraform show -json` output directly |
| NIS2 compliance requires EU jurisdiction proof | 100% EU-owned providers, Sovereign Score in HTML report |

---

## Quick Start

### Download Binary (Recommended)

```bash
# Linux (amd64)
curl -L https://github.com/euroopencost/euroopencost/releases/latest/download/eucost-linux-amd64 -o eucost
chmod +x eucost
./eucost --help

# macOS (Apple Silicon)
curl -L https://github.com/euroopencost/euroopencost/releases/latest/download/eucost-darwin-arm64 -o eucost
chmod +x eucost

# Windows (PowerShell)
Invoke-WebRequest -Uri https://github.com/euroopencost/euroopencost/releases/latest/download/eucost-windows-amd64.exe -OutFile eucost.exe
```

### Build from Source

```bash
git clone https://github.com/euroopencost/euroopencost.git
cd euroopencost
go build -o eucost ./cmd/eucost

# Cross-compile
GOOS=linux   GOARCH=amd64 go build -o eucost-linux-amd64   ./cmd/eucost
GOOS=darwin  GOARCH=arm64 go build -o eucost-darwin-arm64  ./cmd/eucost
GOOS=windows GOARCH=amd64 go build -o eucost-windows-amd64.exe ./cmd/eucost
```

---

## Editions

| Feature | Community (free) | Pro |
|---------|:---:|:---:|
| `eucost plan` → table output | ✓ | ✓ |
| `eucost plan` → JSON output | — | ✓ |
| `eucost plan` → HTML report | — | ✓ |
| `eucost compare` | — | ✓ |
| `eucost breakdown` | — | ✓ |
| `eucost serve` (Web IDE) | — | ✓ |
| `eucost mcp` (AI agent access) | — | ✓ |
| Token required | none | [dashboard.euroopencost.eu](https://dashboard.euroopencost.eu) |

### Activate Pro

```bash
# Store your token (persisted in ~/.eucost/token)
eucost auth login --token <YOUR_TOKEN>

# Check status
eucost auth status

# Remove token (back to Community)
eucost auth logout
```

**CI/CD:** Set the `EUCOST_TOKEN` environment variable — no `auth login` needed:

```yaml
- name: Calculate costs (Pro)
  env:
    EUCOST_TOKEN: ${{ secrets.EUCOST_TOKEN }}
  run: eucost plan plan.json -o json
```

---

## Usage

### 1. Generate a Terraform plan

```bash
cd your-terraform-project
terraform plan -out=plan.tfplan
terraform show -json plan.tfplan > plan.json
```

### 2. Calculate costs

```bash
# Table output (default) — Community & Pro
eucost plan plan.json

# JSON output (for CI/CD pipelines) — Pro
eucost plan plan.json -o json

# HTML report (for management / compliance) — Pro
eucost plan plan.json -o html > report.html

# Read from stdin (pipe-friendly)
terraform show -json plan.tfplan | eucost plan -
```

### 3. Compare providers — Pro

```bash
# Side-by-side cost comparison
eucost compare otc-plan.json hetzner-plan.json

# Output:
# +-------------------+----------+-----------+-----------+
# | DATEI             | PROVIDER | EUR/STUNDE | EUR/MONAT |
# +-------------------+----------+-----------+-----------+
# | otc-plan.json     | otc      |     0.4403 |    317.06 |
# | hetzner-plan.json | hetzner  |     0.0125 |      9.00 * |
# +-------------------+----------+-----------+-----------+
# >> hetzner-plan.json ist 308.06 EUR/Monat guenstiger
```

### 4. Auto-run terraform + calculate (breakdown) — Pro

```bash
# Run terraform plan + cost calculation in one step
eucost breakdown --path ./my-terraform-dir
eucost breakdown --path ./my-terraform-dir -o html > report.html
```

---

## Supported Providers

### OpenTelekomCloud (OTC)

| Terraform Resource | Service | Pricing |
|-------------------|---------|---------|
| `opentelekomcloud_compute_instance_v2` | ECS | Per flavor/hour |
| `opentelekomcloud_blockstorage_volume_v2` | EVS | Per GB/month |
| `opentelekomcloud_vpc_eip_v1` | EIP | Per EIP/hour |
| `opentelekomcloud_lb_loadbalancer_v2` | ELB | Per instance/hour |
| `opentelekomcloud_rds_instance_v3` | RDS | Per flavor/hour |
| `opentelekomcloud_nat_gateway_v2` | NAT | Per spec/hour |
| `opentelekomcloud_dcs_instance_v1` | DCS | Per capacity/hour |
| `opentelekomcloud_obs_bucket` | OBS | Per bucket |
| `opentelekomcloud_cce_cluster_v3` | CCE | Per flavor/hour |
| VPC, Subnet, Security Group, SG Rule | — | Free |

**Pricing source**: [OTC Pricing API](https://calculator.otc-service.com/en/open-telekom-price-api/) — live, no auth required.

### Hetzner Cloud

| Terraform Resource | Service | Pricing |
|-------------------|---------|---------|
| `hcloud_server` | Server | Per server type/hour |
| `hcloud_volume` | Volume | Per GB/month |
| `hcloud_floating_ip` | Floating IP | Per IP/hour |
| `hcloud_load_balancer` | Load Balancer | Per type/hour |
| Firewall, Network, SSH Key | — | Free |

**Pricing source**: Hetzner Cloud API (`HCLOUD_TOKEN` optional — falls back to embedded prices from 2025).

### STACKIT

| Terraform Resource | Service | Pricing |
|-------------------|---------|---------|
| `stackit_server` | Server | Per machine type/hour |
| `stackit_volume` | Volume | Per GB/month |
| `stackit_object_storage_bucket` | OBS | Per bucket |
| Security Group | — | Free |

**Pricing source**: Static prices (no public STACKIT pricing API available).

### IONOS Cloud

| Terraform Resource | Service | Pricing |
|-------------------|---------|---------|
| `ionoscloud_server` | Server | Per cores + RAM/hour |
| `ionoscloud_volume` | Volume | Per GB/month |
| `ionoscloud_ipblock` | IP Block | Per IP/hour |
| Datacenter, LAN, NIC | — | Free |

**Pricing source**: IONOS Cloud API (`IONOS_TOKEN` optional — falls back to embedded prices).

---

## Output Formats

### `--output table` (default)

ASCII table for terminal use. Compatible with PowerShell (CP1252).

### `--output json`

Machine-readable JSON for CI/CD pipelines and integrations:

```json
{
  "resources": [
    {
      "name": "main",
      "provider": "otc",
      "type": "opentelekomcloud_compute_instance_v2",
      "service_name": "ecs",
      "flavor": "s3.medium.4",
      "display_type": "ECS s3.medium.4",
      "hourly_price": "0.0529",
      "monthly_price": "38.09"
    }
  ],
  "total": {
    "hourly_price": "0.4403",
    "monthly_price": "317.06"
  }
}
```

### `--output html`

Self-contained HTML report with the EuroOpenCost design system — ready for CFO reviews
and compliance documentation. Includes:
- EU Sovereign Score ring chart
- Provider split breakdown
- Full resource table with costs
- Generated timestamp

```bash
eucost plan plan.json -o html > report.html
```

---

## CI/CD Integration

> JSON and HTML output require a Pro token. Set `EUCOST_TOKEN` as a repository secret.

### GitHub Actions

```yaml
- name: Calculate Infrastructure Costs (Pro)
  env:
    EUCOST_TOKEN: ${{ secrets.EUCOST_TOKEN }}
  run: |
    terraform plan -out=plan.tfplan
    terraform show -json plan.tfplan | ./eucost plan - -o json > costs.json
    cat costs.json

- name: Upload HTML Cost Report (Pro)
  env:
    EUCOST_TOKEN: ${{ secrets.EUCOST_TOKEN }}
  run: |
    terraform show -json plan.tfplan | ./eucost plan - -o html > cost-report.html

- uses: actions/upload-artifact@v4
  with:
    name: cost-report
    path: cost-report.html
```

### Shell wrapper

```bash
# Use the included wrapper script
./scripts/eucost.sh              # Table output
./scripts/eucost.sh -o json      # JSON output
./scripts/eucost.sh -o html      # HTML report
```

---

## Architecture

```
euroopencost/
├── cmd/eucost/              # CLI entry points (plan, compare, breakdown)
├── internal/
│   ├── models/              # Resource + Total structs
│   ├── parser/              # Terraform plan JSON parser
│   ├── pricing/             # OTC API client + multi-provider calculator
│   │   ├── hetzner/         # Hetzner Cloud pricing
│   │   ├── stackit/         # STACKIT pricing
│   │   └── ionos/           # IONOS Cloud pricing
│   └── renderer/            # Output formats (table, json, html)
├── site/                    # Landing page (euroopencost.eu)
├── scripts/                 # Shell wrappers (eucost.sh, eucost.ps1)
└── testdata/                # Example Terraform plan JSON files
```

### Adding a new provider

1. **Add pricing client**: Create `internal/pricing/<provider>/api.go`
2. **Add parser support**: Extend `internal/parser/terraform.go` with new resource types
3. **Add display names**: Extend `serviceDisplayNames` in `internal/models/resource.go`
4. **Route in calculator**: Add provider case in `internal/pricing/calculator.go`

No other files need to change. The renderer, CLI, and output formats work automatically.

---

## Contributing

Contributions welcome! Especially:

- New EU cloud providers (OVH, Scaleway, Exoscale, UpCloud...)
- Pricing corrections and updates
- Additional resource types for existing providers
- Bug reports and test cases

Please open an issue first to discuss major changes.

---

## License

MIT License — see [LICENSE](LICENSE).

---

*EuroOpenCost is not affiliated with any cloud provider. Prices are estimates based on
public list prices and may not reflect discounts, reserved instances, or special contracts.*
