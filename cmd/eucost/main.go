package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/euroopencost/euroopencost/internal/api"
	"github.com/euroopencost/euroopencost/internal/auth"
	"github.com/euroopencost/euroopencost/internal/models"
	"github.com/euroopencost/euroopencost/internal/parser"
	"github.com/euroopencost/euroopencost/internal/pricing"
	"github.com/euroopencost/euroopencost/internal/pricing/hetzner"
	"github.com/euroopencost/euroopencost/internal/pricing/ionos"
	"github.com/euroopencost/euroopencost/internal/pricing/stackit"
	"github.com/euroopencost/euroopencost/internal/renderer"
	"github.com/euroopencost/euroopencost/pkg/mcp"
	"github.com/euroopencost/euroopencost/pkg/policy"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "eucost",
		Short: "eucost - The open-source cost estimator for European Cloud infrastructure.",
		Long: `EuroOpenCost | Transparent Pricing for Sovereign Clouds.

Berechnet Cloud-Kosten aus einem Terraform Plan JSON fuer europaeische Cloud-Provider.
Unterstuetzte Provider: OTC, Hetzner, STACKIT, IONOS`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	var outputFormat string
	var customerName string

	planCmd := &cobra.Command{
		Use:   "plan <datei>",
		Short: "Kosten aus Terraform Plan JSON berechnen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Pro-Features: json, html, --customer
			if outputFormat != "table" && outputFormat != "" {
				if err := auth.RequirePro(); err != nil {
					return err
				}
			}
			if customerName != "" {
				if err := auth.RequirePro(); err != nil {
					return err
				}
			}
			return runPlan(args[0], outputFormat, customerName)
		},
	}

	planCmd.Flags().StringVarP(&outputFormat, "output", "o", "table", "Output-Format: table (kostenlos), json, html (Pro)")
	planCmd.Flags().StringVar(&customerName, "customer", "", "Name des Kunden fuer den Report (Pro)")
	rootCmd.AddCommand(planCmd)
	rootCmd.AddCommand(newAuthCmd())
	rootCmd.AddCommand(newMCPServerCmd())
	rootCmd.AddCommand(newServeCmd())
	rootCmd.AddCommand(newBreakdownCmd())
	rootCmd.AddCommand(newCompareCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newMCPServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Startet den MCP Server (Model Context Protocol) [Pro]",
		Long:  `EuroOpenCost Layer 3: Ermoeglicht KI-Agenten den Zugriff auf Kosten- und Souveraenitaets-Daten.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.RequirePro(); err != nil {
				return err
			}
			server := &mcp.Server{
				Name:    "EuroOpenCost-MCP",
				Version: "0.1.0",
			}
			fmt.Fprintf(os.Stderr, "MCP Server startet auf stdin/stdout...\n")
			return server.Start(cmd.Context())
		},
	}
}

func newServeCmd() *cobra.Command {
	var port string
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Startet den EuroOpenCost API Server und die Sovereign IDE [Pro]",
		Long:  `EuroOpenCost Layer 2: Bietet ein Web-Interface zur Analyse von Terraform-Plänen.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.RequirePro(); err != nil {
				return err
			}
			// Resolve site/ relative to the binary's location so serve works
			// from any working directory.
			exe, err := os.Executable()
			if err != nil {
				exe = "."
			}
			staticDir := filepath.Join(filepath.Dir(exe), "site")
			// Fallback: if site/ doesn't exist next to binary, try cwd/site
			if _, serr := os.Stat(staticDir); serr != nil {
				staticDir = "site"
			}
			router := &api.Router{
				BaseDomain: "localhost",
				StaticDir:  staticDir,
			}
			addr := ":" + port
			return router.Start(addr)
		},
	}
	cmd.Flags().StringVarP(&port, "port", "p", "8080", "Port für den API Server")
	return cmd
}

// runPlan führt die Kostenberechnung für eine Terraform Plan JSON-Datei aus.
// Dateiname "-" liest von stdin.
func runPlan(filename, outputFormat, customerName string) error {
	// 1. Terraform Plan parsen
	p := parser.NewParser()
	var resources []models.Resource
	var err error

	if filename == "-" {
		// Von stdin lesen
		resources, err = p.ParseReader(os.Stdin)
	} else {
		resources, err = p.Parse(filename)
	}
	if err != nil {
		return fmt.Errorf("Plan parsen fehlgeschlagen: %w", err)
	}

	if len(resources) == 0 {
		fmt.Println("Keine unterstützten Ressourcen gefunden.")
		return nil
	}

	// 2. Preise berechnen (alle Provider)
	calc := pricing.NewCalculator(pricing.NewClient(), hetzner.NewClient(), stackit.NewClient(), ionos.NewClient())
	resources, total, err := calc.Calculate(resources)
	if err != nil {
		return fmt.Errorf("Preisberechnung fehlgeschlagen: %w", err)
	}

	// 2.b Policy Check (Enforcer)
	enforcer := policy.NewEnforcer()
	policyErrs := enforcer.Validate(resources)
	if len(policyErrs) > 0 {
		fmt.Fprintf(os.Stderr, "\n[Policy Alert] Einige Regeln wurden verletzt:\n")
		for _, pErr := range policyErrs {
			fmt.Fprintf(os.Stderr, "  - %v\n", pErr)
		}
		fmt.Fprintf(os.Stderr, "\n")
	}

	// 3. Renderer auswählen
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

	// 4. Kunde setzen
	if customerName != "" {
		r.SetCustomer(customerName)
	}

	// 5. Ausgabe rendern
	if err := r.Render(resources, total); err != nil {
		return fmt.Errorf("Ausgabe fehlgeschlagen: %w", err)
	}

	return nil
}
