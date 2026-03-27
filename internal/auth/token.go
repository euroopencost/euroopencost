package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Validator prüft ob ein Token gültig ist.
// Hinter diesem Interface kann später eine echte API-Validierung eingesetzt werden.
type Validator interface {
	Validate(token string) bool
}

// MVPValidator akzeptiert jeden nicht-leeren Token (Phase 1).
// Wird in Phase 2 durch einen HTTP-basierten Validator ersetzt.
type MVPValidator struct{}

func (v *MVPValidator) Validate(token string) bool {
	return strings.TrimSpace(token) != ""
}

var defaultValidator Validator = &MVPValidator{}

// tokenFilePath gibt den Pfad zur Token-Datei zurück: ~/.eucost/token
func tokenFilePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("Home-Verzeichnis nicht gefunden: %w", err)
	}
	return filepath.Join(home, ".eucost", "token"), nil
}

// GetToken lädt den Token (Priorität: Env-Variable > Datei).
func GetToken() string {
	if t := strings.TrimSpace(os.Getenv("EUCOST_TOKEN")); t != "" {
		return t
	}
	path, err := tokenFilePath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// IsAuthenticated gibt true zurück wenn ein gültiger Token vorhanden ist.
func IsAuthenticated() bool {
	token := GetToken()
	return defaultValidator.Validate(token)
}

// SaveToken speichert den Token in ~/.eucost/token.
func SaveToken(token string) error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("Verzeichnis konnte nicht erstellt werden: %w", err)
	}
	return os.WriteFile(path, []byte(strings.TrimSpace(token)+"\n"), 0600)
}

// RemoveToken löscht den gespeicherten Token.
func RemoveToken() error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Token konnte nicht gelöscht werden: %w", err)
	}
	return nil
}

// RequirePro gibt einen Fehler zurück wenn kein gültiger Token vorhanden ist.
func RequirePro() error {
	if IsAuthenticated() {
		return nil
	}
	return fmt.Errorf(`[eucost] Diese Funktion erfordert einen Pro-Token.

  Jetzt aktivieren: https://dashboard.euroopencost.eu
  Token hinterlegen: eucost auth login --token <TOKEN>`)
}
