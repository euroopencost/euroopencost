package scoring

import (
	"testing"

	"github.com/euroopencost/euroopencost/internal/models"
)

func TestCalculateSovereignScore_ZeroTotal(t *testing.T) {
	info := CalculateSovereignScore(nil, models.Total{MonthlyPrice: 0})
	if info.Score != 100 {
		t.Errorf("got %d, want 100", info.Score)
	}
}

func TestCalculateSovereignScore_PureEU(t *testing.T) {
	resources := []models.Resource{
		{Provider: "hetzner", HourlyPrice: 1.0},
		{Provider: "otc", HourlyPrice: 1.0},
	}
	total := models.Total{MonthlyPrice: 2 * 24 * 30}
	info := CalculateSovereignScore(resources, total)
	if info.Score < 95 {
		t.Errorf("expected score >= 95 for pure EU providers, got %d", info.Score)
	}
	if info.USResources != 0 {
		t.Errorf("expected USResources=0, got %v", info.USResources)
	}
	if info.EUResources == 0 {
		t.Error("expected EUResources > 0")
	}
}

func TestCalculateSovereignScore_PureUS_AWS(t *testing.T) {
	resources := []models.Resource{
		{Provider: "aws", HourlyPrice: 1.0},
	}
	total := models.Total{MonthlyPrice: 1 * 24 * 30}
	info := CalculateSovereignScore(resources, total)
	if info.Score != 0 {
		t.Errorf("expected score=0 for pure AWS, got %d", info.Score)
	}
}

func TestCalculateSovereignScore_Mixed(t *testing.T) {
	resources := []models.Resource{
		{Provider: "hetzner", HourlyPrice: 1.0},
		{Provider: "aws", HourlyPrice: 1.0},
	}
	total := models.Total{MonthlyPrice: 2 * 24 * 30}
	info := CalculateSovereignScore(resources, total)
	if info.Score < 0 || info.Score > 100 {
		t.Errorf("score out of range [0,100]: got %d", info.Score)
	}
	if info.EUResources == 0 {
		t.Error("expected EUResources > 0")
	}
	if info.USResources == 0 {
		t.Error("expected USResources > 0")
	}
}

func TestCalculateSovereignScore_ScoreNeverBelowZero(t *testing.T) {
	resources := []models.Resource{
		{Provider: "gcp", HourlyPrice: 10.0},
	}
	total := models.Total{MonthlyPrice: 10 * 24 * 30}
	info := CalculateSovereignScore(resources, total)
	if info.Score < 0 {
		t.Errorf("score clamping failed: got %d (must be >= 0)", info.Score)
	}
}

func TestCalculateSovereignScore_ScoreNeverAbove100(t *testing.T) {
	resources := []models.Resource{
		{Provider: "stackit", HourlyPrice: 5.0},
	}
	total := models.Total{MonthlyPrice: 5 * 24 * 30}
	info := CalculateSovereignScore(resources, total)
	if info.Score > 100 {
		t.Errorf("score clamping failed: got %d (must be <= 100)", info.Score)
	}
}

func TestCalculateSovereignScore_UnknownProvider(t *testing.T) {
	// Unknown provider treated as non-EU risk=0.5
	resources := []models.Resource{
		{Provider: "some-unknown-cloud", HourlyPrice: 1.0},
	}
	total := models.Total{MonthlyPrice: 1 * 24 * 30}
	info := CalculateSovereignScore(resources, total)
	if info.Score < 0 || info.Score > 100 {
		t.Errorf("score out of range [0,100] for unknown provider: got %d", info.Score)
	}
}
