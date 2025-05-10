package backend

import (
	"context"
	"fmt"

	"github.com/kream404/spoof/models"
	"github.com/kream404/spoof/services/csv"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx context.Context
}

func NewApp() *App {
	return &App{}
}

func (a *App) GenerateCSV() {
	// Trigger the CSV generation logic
	config := models.FileConfig{
		// Fill with required fields
	}

	err := csv.GenerateCSV(config, "./output/output.csv")
	if err != nil {
		runtime.EventsEmit(a.ctx, "csv:generated", fmt.Sprintf("Error: %v", err))
	} else {
		runtime.EventsEmit(a.ctx, "csv:generated", "CSV generation complete!")
	}
}
