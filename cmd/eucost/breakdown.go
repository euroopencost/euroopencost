package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/euroopencost/euroopencost/internal/auth"
	"github.com/euroopencost/euroopencost/internal/parser"
	"github.com/euroopencost/euroopencost/internal/pricing"
	"github.com/euroopencost/euroopencost/internal/pricing/hetzner"
	"github.com/euroopencost/euroopencost/internal/pricing/ionos"
	"github.com/euroopencost/euroopencost/internal/pricing/stackit"
	"github.com/euroopencost/euroopencost/internal/renderer"
	"github.com/spf13/cobra"
)

// newBreakdownCmd erstellt den breakdown Subcommand.
func newBreakdownCmd() *cobra.Command {
	var outputFormat string
	var path string

	cmd := &cobra.Command{
		Use:   "breakdown",
		Short: "Terraform Plan automatisch ausführen und Kosten berechnen",
		Long: `Fuehrt 'terraform plan' im angegebenen Verzeichnis aus,
liest den Plan und berechnet die Cloud-Kosten - alles in einem Schritt.

Benoetigt: terraform installiert + Provider Credentials als Umgebungsvariablen.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.RequirePro(); err != nil {
				return err
			}
			return runBreakdown(path, outputFormat)
		},
	}

	cmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output-Format: table, json, html [Pro]")
	cmd.Flags().StringVar(&path, "path", ".", "Pfad zum Terraform-Verzeichnis")

	return cmd
}

// runBreakdown führt terraform plan aus und berechnet die Kosten.
func runBreakdown(tfPath, outputFormat string) error {
	// Absoluten Pfad auflösen
	absPath, err := filepath.Abs(tfPath)
	if err != nil {
		return fmt.Errorf("Pfad auflösen fehlgeschlagen: %w", err)
	}

	// Prüfen ob terraform installiert ist
	if _, err := exec.LookPath("terraform"); err != nil {
		return fmt.Errorf("terraform nicht gefunden — bitte installieren: https://developer.hashicorp.com/terraform/install")
	}

	fmt.Fprintf(os.Stderr, ">>terraform plan wird ausgeführt in: %s\n", absPath)

	// Temp-Datei für den Plan
	tmpPlan := filepath.Join(absPath, ".eucost-tmp.tfplan")
	defer os.Remove(tmpPlan)

	// terraform plan -out=.otc-cost-tmp.tfplan ausführen
	planCmd := exec.Command("terraform", "plan", "-out="+tmpPlan)
	planCmd.Dir = absPath
	planCmd.Stdout = os.Stdout
	planCmd.Stderr = os.Stderr

	if err := planCmd.Run(); err != nil {
		return fmt.Errorf("terraform plan fehlgeschlagen: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\n>>Kosten werden berechnet...\n\n")

	// terraform show -json ausführen und stdout einfangen
	showCmd := exec.Command("terraform", "show", "-json", tmpPlan)
	showCmd.Dir = absPath
	showCmd.Stderr = os.Stderr

	var planJSON bytes.Buffer
	showCmd.Stdout = &planJSON

	if err := showCmd.Run(); err != nil {
		return fmt.Errorf("terraform show fehlgeschlagen: %w", err)
	}

	// Plan JSON parsen
	p := parser.NewParser()
	resources, err := p.ParseReader(&planJSON)
	if err != nil {
		return fmt.Errorf("Plan parsen fehlgeschlagen: %w", err)
	}

	if len(resources) == 0 {
		fmt.Println("Keine unterstützten Ressourcen gefunden.")
		return nil
	}

	// Preise berechnen (alle Provider)
	calc := pricing.NewCalculator(pricing.NewClient(), hetzner.NewClient(), stackit.NewClient(), ionos.NewClient())
	resources, total, err := calc.Calculate(resources)
	if err != nil {
		return fmt.Errorf("Preisberechnung fehlgeschlagen: %w", err)
	}

	// Renderer auswählen
	var r renderer.Renderer
	switch outputFormat {
	case "table", "":
		r = renderer.NewTableRenderer()
	case "json":
		r = renderer.NewJSONRenderer()
	case "html":
		r = renderer.NewHTMLRenderer()
	default:
		return fmt.Errorf("Unbekanntes Output-Format: '%s' (unterstützt: table, json, html)", outputFormat)
	}

	// Ausgabe rendern
	if err := r.Render(resources, total); err != nil {
		return fmt.Errorf("Ausgabe fehlgeschlagen: %w", err)
	}

	return nil
}
