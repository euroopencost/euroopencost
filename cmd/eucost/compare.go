package main

import (
	"fmt"
	"os"

	"github.com/euroopencost/euroopencost/internal/auth"
	"github.com/euroopencost/euroopencost/internal/models"
	"github.com/euroopencost/euroopencost/internal/parser"
	"github.com/euroopencost/euroopencost/internal/pricing"
	"github.com/euroopencost/euroopencost/internal/pricing/hetzner"
	"github.com/euroopencost/euroopencost/internal/pricing/ionos"
	"github.com/euroopencost/euroopencost/internal/pricing/stackit"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// newCompareCmd erstellt den compare Subcommand.
func newCompareCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compare <datei1> <datei2> [datei3...]",
		Short: "Kosten mehrerer Terraform Plans nebeneinander vergleichen [Pro]",
		Long: `Vergleicht die Gesamtkosten mehrerer Terraform Plan JSON-Dateien.
Nützlich um Kosten zwischen verschiedenen Cloud-Anbietern oder Umgebungen zu vergleichen.

Beispiel:
  eucost compare testdata/otc-plan.json testdata/hetzner-plan.json`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.RequirePro(); err != nil {
				return err
			}
			return runCompare(args)
		},
	}
	return cmd
}

// planResult enthält das Ergebnis einer einzelnen Plan-Auswertung.
type planResult struct {
	Filename  string
	Provider  string
	Resources []models.Resource
	Total     models.Total
	Error     error
}

// runCompare wertet alle übergebenen Plan-Dateien aus und zeigt sie nebeneinander.
func runCompare(files []string) error {
	calc := pricing.NewCalculator(pricing.NewClient(), hetzner.NewClient(), stackit.NewClient(), ionos.NewClient())
	p := parser.NewParser()

	results := make([]planResult, 0, len(files))

	// Jeden Plan einzeln auswerten
	for _, file := range files {
		res, err := p.Parse(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warnung: '%s' übersprungen: %v\n", file, err)
			continue
		}

		priced, total, err := calc.Calculate(res)
		provider := detectProvider(res)

		results = append(results, planResult{
			Filename:  file,
			Provider:  provider,
			Resources: priced,
			Total:     total,
			Error:     err,
		})
	}

	if len(results) == 0 {
		return fmt.Errorf("keine gültigen Plan-Dateien gefunden")
	}

	renderCompare(results)
	return nil
}

// detectProvider erkennt den dominanten Provider in einer Ressourcenliste.
func detectProvider(resources []models.Resource) string {
	counts := make(map[string]int)
	for _, r := range resources {
		if r.Provider != "" {
			counts[r.Provider]++
		}
	}

	dominant := "unbekannt"
	max := 0
	for p, count := range counts {
		if count > max {
			max = count
			dominant = p
		}
	}
	return dominant
}

// renderCompare gibt die Vergleichstabelle aus.
func renderCompare(results []planResult) {
	// Überschriften bauen
	headers := []string{"Datei", "Provider", "EUR/Stunde", "EUR/Monat"}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(true)
	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_CENTER,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
	})

	var cheapest *planResult
	for i := range results {
		r := &results[i]
		if cheapest == nil || r.Total.MonthlyPrice < cheapest.Total.MonthlyPrice {
			cheapest = r
		}
	}

	for i := range results {
		r := &results[i]
		monthlyStr := fmt.Sprintf("%.2f", r.Total.MonthlyPrice)
		hourlyStr := fmt.Sprintf("%.4f", r.Total.HourlyPrice)

		// Günstigsten Provider markieren
		if r == cheapest && len(results) > 1 {
			monthlyStr = monthlyStr + " *"
		}

		table.Append([]string{
			r.Filename,
			r.Provider,
			hourlyStr,
			monthlyStr,
		})
	}

	table.Render()

	// Ersparnis anzeigen wenn 2 Plans verglichen werden
	if len(results) == 2 {
		diff := results[0].Total.MonthlyPrice - results[1].Total.MonthlyPrice
		if diff > 0 {
			fmt.Printf("\n>>%s ist %.2f €/Monat günstiger als %s\n",
				results[1].Filename, diff, results[0].Filename)
		} else if diff < 0 {
			fmt.Printf("\n>>%s ist %.2f €/Monat günstiger als %s\n",
				results[0].Filename, -diff, results[1].Filename)
		} else {
			fmt.Println("\nBeide Plans kosten gleich viel.")
		}
	}
}
