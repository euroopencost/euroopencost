package renderer

import "github.com/euroopencost/euroopencost/internal/models"

// Renderer ist das Interface für alle Output-Formate.
type Renderer interface {
	Render(resources []models.Resource, total models.Total) error
	Name() string
	SetCustomer(name string)
}
