package renderer

import (
	"fmt"
	"os"

	"github.com/euroopencost/euroopencost/internal/models"
	"github.com/euroopencost/euroopencost/internal/scoring"
	"github.com/olekukonko/tablewriter"
)

// TableRenderer gibt die Ressourcen als formatierte ASCII-Tabelle aus.
type TableRenderer struct {
	customer string
}

// NewTableRenderer erstellt einen neuen TableRenderer.
func NewTableRenderer() *TableRenderer {
	return &TableRenderer{}
}

// Name gibt den Namen des Renderers zurück.
func (r *TableRenderer) Name() string {
	return "table"
}

// SetCustomer setzt den Kundennamen für den Report.
func (r *TableRenderer) SetCustomer(name string) {
	r.customer = name
}

// Render gibt alle Ressourcen und die Gesamtsumme als Tabelle aus.
func (r *TableRenderer) Render(resources []models.Resource, total models.Total) error {
	if r.customer != "" {
		fmt.Printf("EuroOpenCost Report for: %s\n\n", r.customer)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Ressource", "Typ", "EUR/Stunde", "EUR/Monat"})
	table.SetBorder(true)
	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_LEFT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
	})

	for _, res := range resources {
		table.Append([]string{
			res.Name,
			res.DisplayType(), // SERVICE + Flavor kombiniert
			fmt.Sprintf("%.4f", res.HourlyPrice),
			fmt.Sprintf("%.2f", res.MonthlyPrice()),
		})
	}

	// Trennlinie vor TOTAL
	table.SetFooter([]string{
		"TOTAL",
		"",
		fmt.Sprintf("%.4f", total.HourlyPrice),
		fmt.Sprintf("%.2f", total.MonthlyPrice),
	})

	table.Render()
	fmt.Println()

	// Sovereign Score
	sov := scoring.CalculateSovereignScore(resources, total)
	fmt.Printf("   SOVEREIGN SCORE: %d%%  ($S$ Index: Unabhängigkeit & Compliance)\n", sov.Score)
	if sov.Score >= 90 {
		fmt.Printf("   [Sovereign Cloud Status: EXCELLENT (EU-Souverän)]\n")
	} else if sov.Score >= 50 {
		fmt.Printf("   [Sovereign Cloud Status: MODERATE (Mischumgebung)]\n")
	} else {
		fmt.Printf("   [Sovereign Cloud Status: LOW (US-Abhängigkeit erkannt)]\n")
	}
	fmt.Println()

	return nil
}
