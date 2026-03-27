package renderer

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"
	"time"

	"github.com/euroopencost/euroopencost/internal/models"
	"github.com/euroopencost/euroopencost/internal/scoring"
)

// HTMLRenderer generiert einen self-contained HTML-Report im EuroOpenCost Design.
type HTMLRenderer struct {
	customer string
}

// NewHTMLRenderer erstellt einen neuen HTMLRenderer.
func NewHTMLRenderer() *HTMLRenderer {
	return &HTMLRenderer{}
}

// Name gibt den Namen des Renderers zurück.
func (r *HTMLRenderer) Name() string {
	return "html"
}

// SetCustomer setzt den Kundennamen für den Report.
func (r *HTMLRenderer) SetCustomer(name string) {
	r.customer = name
}

// providerEntry fasst Provider-Kostenanteile zusammen.
type providerEntry struct {
	Name        string
	DisplayName string
	Monthly     float64
	Percent     int
	Color       string
	BgColor     string
	TextColor   string
}

// templateData enthält alle Daten für das HTML-Template.
type templateData struct {
	GeneratedAt   string
	CustomerName  string
	TotalMonthly  string
	TotalYearly   string
	TotalHourly   string
	Providers     []providerEntry
	Resources     []models.Resource
	Sovereign     scoring.SovereignInfo
	ResourceCount int
	CO2           string // Placeholder: 42.8 g/kWh
}

// providerDisplayNames mappt Provider-IDs auf Anzeigenamen.
var providerDisplayNames = map[string]string{
	"otc":     "OpenTelekomCloud",
	"hetzner": "Hetzner Cloud",
	"stackit": "STACKIT",
	"ionos":   "IONOS Cloud",
}

// providerColors mappt Provider-IDs auf Design-Farben.
var providerColors = map[string][3]string{
	// {stroke-color/primary-fill, bg-color, text-color}
	"otc":     {"#003399", "#dce1ff", "#00164e"},
	"hetzner": {"#fecb00", "#ffe08b", "#241a00"},
	"stackit": {"#83fc8e", "#004815", "#ffffff"},
	"ionos":   {"#b5c4ff", "#002068", "#ffffff"},
}

// Render schreibt den HTML-Report nach stdout.
func (r *HTMLRenderer) Render(resources []models.Resource, total models.Total) error {
	// Provider-Split berechnen
	providerMonthly := make(map[string]float64)
	for _, res := range resources {
		providerMonthly[res.Provider] += res.MonthlyPrice()
	}

	var providers []providerEntry
	for id, monthly := range providerMonthly {
		displayName, ok := providerDisplayNames[id]
		if !ok {
			displayName = strings.ToUpper(id)
		}
		pct := 0
		if total.MonthlyPrice > 0 {
			pct = int(monthly / total.MonthlyPrice * 100)
		}
		colors, ok := providerColors[id]
		if !ok {
			colors = [3]string{"#747684", "#e2e2e2", "#1a1c1c"}
		}
		providers = append(providers, providerEntry{
			Name:        id,
			DisplayName: displayName,
			Monthly:     monthly,
			Percent:     pct,
			Color:       colors[0],
			BgColor:     colors[1],
			TextColor:   colors[2],
		})
	}

	// Sovereign Score berechnen
	sov := scoring.CalculateSovereignScore(resources, total)

	data := templateData{
		GeneratedAt:   time.Now().Format("2006-01-02 15:04:05"),
		CustomerName:  r.customer,
		TotalMonthly:  fmt.Sprintf("%.2f", total.MonthlyPrice),
		TotalYearly:   fmt.Sprintf("%.2f", total.MonthlyPrice*12),
		TotalHourly:   fmt.Sprintf("%.4f", total.HourlyPrice),
		Providers:     providers,
		Resources:     resources,
		Sovereign:     sov,
		ResourceCount: len(resources),
		CO2:           "42.8", // Placeholder
	}

	tmpl, err := template.New("report").Funcs(template.FuncMap{
		"monthlyPrice": func(r models.Resource) string {
			return fmt.Sprintf("%.2f", r.MonthlyPrice())
		},
		"hourlyPrice": func(r models.Resource) string {
			return fmt.Sprintf("%.4f", r.HourlyPrice)
		},
		"displayType": func(r models.Resource) string {
			return r.DisplayType()
		},
		"providerLabel": func(p string) string {
			if name, ok := providerDisplayNames[p]; ok {
				return name
			}
			return strings.ToUpper(p)
		},
		"isFree": func(r models.Resource) bool {
			return r.HourlyPrice == 0
		},
		"dashoffset": func(pct int) float64 {
			// SVG stroke-dashoffset für Ring-Chart (circumference = 282.7 für r=45)
			return 282.7 * (1 - float64(pct)/100.0)
		},
		"contains": func(s, substr string) bool {
			return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
		},
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("HTML template parse fehlgeschlagen: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("HTML template render fehlgeschlagen: %w", err)
	}

	_, err = os.Stdout.Write(buf.Bytes())
	return err
}

// htmlTemplate ist das eingebettete HTML-Template für den Report.
const htmlTemplate = `<!DOCTYPE html>
<html class="light" lang="en">
<head>
    <meta charset="utf-8"/>
    <meta content="width=device-width, initial-scale=1.0" name="viewport"/>
    <title>eucost | CFO FinOps Dashboard</title>
    <script src="https://cdn.tailwindcss.com?plugins=forms,container-queries"></script>
    <link href="https://fonts.googleapis.com/css2?family=Manrope:wght@400;500;700;800&family=Inter:wght@400;500;600&display=swap" rel="stylesheet"/>
    <link href="https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined:wght,FILL@100..700,0..1&display=swap" rel="stylesheet"/>
    <script>
      tailwind.config = {
        darkMode: "class",
        theme: {
          extend: {
            colors: {
              "primary-fixed-dim": "#b5c4ff",
              "surface-container-lowest": "#ffffff",
              "tertiary-container": "#004815",
              "primary-fixed": "#dce1ff",
              "inverse-primary": "#b5c4ff",
              "tertiary-fixed-dim": "#66df75",
              "surface-container": "#eeeeee",
              "surface-bright": "#f9f9f9",
              "surface-container-high": "#e8e8e8",
              "outline": "#747684",
              "tertiary": "#002f0b",
              "on-surface-variant": "#444653",
              "tertiary-fixed": "#83fc8e",
              "primary": "#002068",
              "secondary": "#745b00",
              "on-primary-container": "#8aa4ff",
              "on-surface": "#1a1c1c",
              "background": "#f9f9f9",
              "on-background": "#1a1c1c",
              "on-error": "#ffffff",
              "inverse-surface": "#2f3131",
              "on-tertiary-fixed-variant": "#00531a",
              "secondary-fixed": "#ffe08b",
              "on-primary": "#ffffff",
              "surface-container-highest": "#e2e2e2",
              "on-tertiary-container": "#45bf59",
              "on-secondary-fixed-variant": "#584400",
              "on-secondary-fixed": "#241a00",
              "primary-container": "#003399",
              "error-container": "#ffdad6",
              "surface-tint": "#3557bc",
              "on-primary-fixed": "#00164e",
              "on-tertiary-fixed": "#002106",
              "surface": "#f9f9f9",
              "on-secondary-container": "#6e5700",
              "inverse-on-surface": "#f0f1f1",
              "on-tertiary": "#ffffff",
              "surface-container-low": "#f3f3f3",
              "outline-variant": "#c4c5d5",
              "secondary-container": "#fecb00",
              "surface-dim": "#dadada",
              "surface-variant": "#e2e2e2",
              "on-secondary": "#ffffff",
              "secondary-fixed-dim": "#f1c100",
              "error": "#ba1a1a",
              "on-error-container": "#93000a",
              "on-primary-fixed-variant": "#153ea3"
            },
            fontFamily: {
              "headline": ["Manrope", "sans-serif"],
              "body": ["Inter", "sans-serif"],
              "label": ["Inter", "sans-serif"]
            },
            borderRadius: {"DEFAULT": "0.125rem", "lg": "0.25rem", "xl": "0.5rem", "full": "0.75rem"},
          },
        },
      }
    </script>
    <style>
        .material-symbols-outlined { font-variation-settings: 'FILL' 0, 'wght' 400, 'GRAD' 0, 'opsz' 24; }
        .compliance-ribbon { width: 4px; height: 100%; position: absolute; left: 0; top: 0; }
        .glass-panel { background: rgba(255, 255, 255, 0.85); backdrop-filter: blur(20px); }
        .tabular-numbers { font-variant-numeric: tabular-nums; }
        body { min-height: 100dvh; }
    </style>
</head>
<body class="bg-surface font-body text-on-surface selection:bg-primary-fixed selection:text-on-primary-fixed">

<header class="fixed top-0 w-full z-50 flex justify-between items-center px-6 py-4 glass-panel border-b border-surface-container-high">
    <div class="flex items-center gap-4">
        <div class="w-10 h-10 rounded-full bg-primary-container flex items-center justify-center overflow-hidden">
            <span class="material-symbols-outlined text-white">account_balance</span>
        </div>
        <span class="text-xl font-bold tracking-tighter text-primary font-headline uppercase">eucost</span>
    </div>
    <nav class="hidden md:flex gap-8">
        <a class="text-primary font-semibold font-manrope text-sm tracking-tight" href="#">Dashboard</a>
        <a class="text-on-surface-variant hover:text-primary transition-colors font-manrope text-sm tracking-tight" href="#">Reports</a>
        <a class="text-on-surface-variant hover:text-primary transition-colors font-manrope text-sm tracking-tight" href="#">Compliance</a>
    </nav>
    <div class="flex items-center gap-3 bg-tertiary-fixed text-on-tertiary-fixed px-3 py-1.5 rounded-full text-xs font-bold tracking-tight">
        <span class="material-symbols-outlined text-[16px]" style="font-variation-settings: 'FILL' 1;">verified_user</span>
        {{.Sovereign.Score}}% Sovereign
    </div>
</header>

<main class="pt-32 pb-32 px-6 md:px-12 max-w-7xl mx-auto space-y-16">
    <!-- Hero Section -->
    <section class="grid grid-cols-1 lg:grid-cols-12 gap-8 items-end">
        <div class="lg:col-span-7 space-y-6">
            <p class="text-on-surface-variant font-label text-sm uppercase tracking-widest">{{if .CustomerName}}{{.CustomerName}}{{else}}Institutional Ledger v4.2{{end}}</p>
            <h1 class="text-5xl md:text-7xl font-headline font-extrabold tracking-tighter leading-tight text-primary">
                Truly EU <br/>Sovereign Score.
            </h1>
            <p class="text-on-surface-variant text-lg max-w-xl">
                Infrastructure alignment with European digital sovereignty. Ensuring your operational expenditure stays within EU jurisdiction.
            </p>
        </div>
        <div class="lg:col-span-5 flex flex-col items-center lg:items-end">
            <div class="relative w-48 h-48 md:w-56 md:h-56">
                <svg class="w-full h-full -rotate-90" viewbox="0 0 100 100">
                    <circle class="text-surface-container-highest" cx="50" cy="50" fill="none" r="45" stroke="currentColor" stroke-width="8"></circle>
                    <circle class="text-on-tertiary-container" cx="50" cy="50" fill="none" r="45" stroke="currentColor" stroke-dasharray="282.7" stroke-dashoffset="{{dashoffset .Sovereign.Score}}" stroke-width="8"></circle>
                </svg>
                <div class="absolute inset-0 flex flex-col items-center justify-center">
                    <span class="text-5xl font-headline font-extrabold tabular-numbers text-primary">{{.Sovereign.Score}}%</span>
                    <span class="text-xs font-label font-bold text-on-tertiary-container uppercase tracking-tighter">Compliant</span>
                </div>
            </div>
        </div>
    </section>

    <!-- Bento Grid -->
    <section class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
        <!-- Financial Overview -->
        <div class="relative bg-surface-container-lowest p-8 rounded-xl shadow-[0_20px_40px_rgba(0,32,104,0.04)] overflow-hidden group">
            <div class="compliance-ribbon bg-primary-container"></div>
            <div class="flex justify-between items-start mb-8">
                <span class="font-label font-bold text-xs uppercase tracking-widest text-on-surface-variant">Estimated Yearly Cost</span>
                <span class="material-symbols-outlined text-primary">payments</span>
            </div>
            <div class="space-y-1">
                <p class="text-4xl font-headline font-semibold tabular-numbers text-primary">€ {{.TotalYearly}}</p>
                <p class="text-sm text-on-surface-variant font-medium">Monthly: € {{.TotalMonthly}}</p>
            </div>
            <div class="mt-6 flex items-center gap-2 text-primary font-bold text-sm">
                <span class="material-symbols-outlined text-sm">euro</span>
                <span>{{.ResourceCount}} active units</span>
            </div>
        </div>

        <!-- Provider Split -->
        <div class="relative bg-surface-container-lowest p-8 rounded-xl shadow-[0_20px_40px_rgba(0,32,104,0.04)] overflow-hidden">
            <div class="compliance-ribbon bg-secondary-container"></div>
            <div class="flex justify-between items-start mb-6">
                <span class="font-label font-bold text-xs uppercase tracking-widest text-on-surface-variant">Provider Distribution</span>
                <span class="material-symbols-outlined text-secondary">analytics</span>
            </div>
            <div class="space-y-4">
                {{range .Providers}}
                <div class="space-y-1">
                    <div class="flex justify-between text-xs font-bold uppercase tracking-tight">
                        <span class="text-on-surface-variant">{{.DisplayName}}</span>
                        <span class="text-primary tabular-numbers">{{.Percent}}%</span>
                    </div>
                    <div class="h-1 w-full bg-surface-container-highest overflow-hidden">
                        <div class="h-full" style="width: {{.Percent}}%; background-color: {{.Color}};"></div>
                    </div>
                </div>
                {{end}}
            </div>
        </div>

        <!-- Savings / ESG -->
        <div class="relative bg-surface-container-lowest p-8 rounded-xl shadow-[0_20px_40px_rgba(0,32,104,0.04)] overflow-hidden">
            <div class="compliance-ribbon bg-tertiary-fixed"></div>
            <div class="flex justify-between items-start mb-8">
                <span class="font-label font-bold text-xs uppercase tracking-widest text-on-surface-variant">Environmental ESG</span>
                <span class="material-symbols-outlined text-on-tertiary-container">eco</span>
            </div>
            <div class="space-y-2">
                <p class="text-4xl font-headline font-semibold tabular-numbers text-primary">{{.CO2}}</p>
                <p class="text-xs font-bold text-on-surface-variant uppercase tracking-widest">g/kWh CO2 Intensity</p>
            </div>
            <div class="mt-6 flex flex-wrap gap-2">
                <span class="bg-tertiary-fixed text-on-tertiary-fixed px-2 py-1 rounded text-[10px] font-bold">CARBON NEUTRAL</span>
            </div>
        </div>
    </section>

    <!-- Data Deep Dive (Resource Inventory) -->
    <section class="space-y-8">
        <h2 class="text-2xl font-headline font-bold text-primary">Resource Inventory</h2>
        <div class="bg-surface-container-lowest rounded-xl shadow-[0_20px_40px_rgba(0,32,104,0.04)] overflow-hidden">
            <div class="overflow-x-auto">
                <table class="w-full text-left border-collapse">
                    <thead>
                        <tr class="bg-surface-container-low text-on-surface-variant text-[10px] font-bold uppercase tracking-[0.2em]">
                            <th class="px-8 py-4">Resource</th>
                            <th class="px-8 py-4">Type</th>
                            <th class="px-8 py-4">Provider</th>
                            <th class="px-8 py-4 text-right">Price/m</th>
                        </tr>
                    </thead>
                    <tbody class="divide-y divide-surface-container-high">
                        {{range .Resources}}
                        <tr class="hover:bg-surface-container-low transition-colors group">
                            <td class="px-8 py-6">
                                <div class="flex items-center gap-4">
                                    <span class="material-symbols-outlined text-outline group-hover:text-primary transition-colors">
                                        {{if contains .Type "server"}}memory{{else if contains .Type "volume"}}database{{else if contains .Type "vpc"}}dns{{else}}deployed_code{{end}}
                                    </span>
                                    <span class="font-bold text-primary">{{.Name}}</span>
                                </div>
                            </td>
                            <td class="px-8 py-6 text-sm text-on-surface-variant">{{displayType .}}</td>
                            <td class="px-8 py-6">
                                <span class="px-2 py-1 rounded text-[10px] font-bold uppercase tracking-tight" style="background-color: {{if eq .Provider "otc"}}#dce1ff{{else if eq .Provider "hetzner"}}#ffe08b{{else}}#eeeeee{{end}}">
                                    {{providerLabel .Provider}}
                                </span>
                            </td>
                            <td class="px-8 py-6 text-right font-headline font-bold text-primary tabular-numbers">
                                € {{monthlyPrice .}}
                            </td>
                        </tr>
                        {{end}}
                    </tbody>
                </table>
            </div>
        </div>
    </section>
</main>

<footer class="bg-primary text-on-primary py-12 px-6 md:px-12 mt-24">
    <div class="max-w-7xl mx-auto flex flex-col md:flex-row justify-between items-center gap-8">
        <div class="flex items-center gap-4">
            <span class="text-2xl font-headline font-extrabold tracking-tighter uppercase">eucost</span>
            <span class="text-xs opacity-60 uppercase tracking-widest border-l border-on-primary/20 pl-4">Sovereign Cloud Intelligence</span>
        </div>
        <div class="text-[10px] uppercase tracking-[0.2em] opacity-60">
            Generated at {{.GeneratedAt}} | © 2026 EuroOpenCost
        </div>
    </div>
</footer>

</body>
</html>`
