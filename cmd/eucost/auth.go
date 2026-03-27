package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/euroopencost/euroopencost/internal/auth"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Pro-Token verwalten",
		Long:  `Verwaltet den EuroOpenCost Pro-Token für erweiterte Funktionen.`,
	}
	authCmd.AddCommand(newAuthLoginCmd())
	authCmd.AddCommand(newAuthLogoutCmd())
	authCmd.AddCommand(newAuthStatusCmd())
	return authCmd
}

func newAuthLoginCmd() *cobra.Command {
	var token string
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Pro-Token hinterlegen",
		Long: `Speichert den Pro-Token lokal unter ~/.eucost/token.

Token erhalten unter: https://dashboard.euroopencost.eu`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if token == "" {
				// Interaktiv fragen wenn kein Flag gesetzt
				fmt.Print("Pro-Token eingeben: ")
				reader := bufio.NewReader(os.Stdin)
				input, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("Eingabe fehlgeschlagen: %w", err)
				}
				token = strings.TrimSpace(input)
			}
			if token == "" {
				return fmt.Errorf("Kein Token eingegeben.")
			}
			if err := auth.SaveToken(token); err != nil {
				return fmt.Errorf("Token konnte nicht gespeichert werden: %w", err)
			}
			fmt.Println("[eucost] Token erfolgreich gespeichert. Pro-Features sind jetzt aktiv.")
			return nil
		},
	}
	cmd.Flags().StringVar(&token, "token", "", "Pro-Token (alternativ: Env-Variable EUCOST_TOKEN)")
	return cmd
}

func newAuthLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Gespeicherten Token entfernen",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.RemoveToken(); err != nil {
				return err
			}
			fmt.Println("[eucost] Token entfernt. Community-Modus aktiv.")
			return nil
		},
	}
}

func newAuthStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Aktuellen Token-Status anzeigen",
		RunE: func(cmd *cobra.Command, args []string) error {
			token := auth.GetToken()
			if token == "" {
				fmt.Println("[eucost] Status: Community (kein Token)")
				fmt.Println("         Token aktivieren: eucost auth login --token <TOKEN>")
				fmt.Println("         Dashboard:        https://dashboard.euroopencost.eu")
				return nil
			}
			// Token maskieren für die Anzeige
			masked := maskToken(token)
			fmt.Printf("[eucost] Status: Pro aktiv\n")
			fmt.Printf("         Token:  %s\n", masked)
			return nil
		},
	}
}

// maskToken zeigt nur die ersten 4 und letzten 4 Zeichen des Tokens.
func maskToken(token string) string {
	if len(token) <= 8 {
		return strings.Repeat("*", len(token))
	}
	return token[:4] + strings.Repeat("*", len(token)-8) + token[len(token)-4:]
}
