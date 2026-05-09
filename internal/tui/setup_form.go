package tui

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/nawodyaishan/mcp-config-tui/internal/config"
	"github.com/nawodyaishan/mcp-config-tui/internal/exa"
)

type setupForm struct {
	form          *huh.Form
	ctx           *wizardContext
	rawKeys       string
	selectedSlice []config.AppID
}

func newSetupForm(ctx *wizardContext, initialRaw string) *setupForm {
	sf := &setupForm{
		ctx:     ctx,
		rawKeys: initialRaw,
	}

	// Initialize selectedSlice from context
	for id, isSelected := range ctx.selected {
		if isSelected {
			sf.selectedSlice = append(sf.selectedSlice, id)
		}
	}

	options := make([]huh.Option[config.AppID], len(ctx.manager.Apps))
	for i, appConfig := range ctx.manager.Apps {
		options[i] = huh.NewOption(appConfig.Name, appConfig.ID)
	}

	groups := []*huh.Group{
		huh.NewGroup(
			huh.NewMultiSelect[config.AppID]().
				Title("Select Target Apps").
				Description("Choose which applications to update with Exa MCP").
				Options(options...).
				Value(&sf.selectedSlice),
		),
	}

	if len(ctx.keys) == 0 {
		groups = append(groups, huh.NewGroup(
			huh.NewText().
				Title("Exa API Keys").
				Description("Paste one or more UUID-style keys (one per line or key = \"...\" format)").
				Placeholder("11111111-2222-3333-4444-555555555555").
				Value(&sf.rawKeys).
				Validate(func(s string) error {
					keys, err := exa.ParseKeys(s)
					if err != nil {
						return fmt.Errorf("invalid keys: %w", err)
					}
					if len(keys) == 0 {
						return fmt.Errorf("at least one valid Exa API key is required")
					}
					return nil
				}),
		))
	}

	sf.form = huh.NewForm(groups...).WithTheme(huh.ThemeCatppuccin())

	return sf
}

func (sf *setupForm) syncToContext() {
	// Sync selected apps
	newSelected := make(map[config.AppID]bool)
	for _, id := range sf.selectedSlice {
		newSelected[id] = true
	}
	sf.ctx.selected = newSelected

	// Sync keys
	keys, _ := exa.ParseKeys(sf.rawKeys)
	sf.ctx.keys = keys
}
