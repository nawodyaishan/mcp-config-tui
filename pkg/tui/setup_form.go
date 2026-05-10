package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/config"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/provider"
	"github.com/nawodyaishan/universal-mcp-sync/pkg/redact"
)

type setupForm struct {
	form             *huh.Form
	ctx              *wizardContext
	selectedProvider string
	credentialValues map[string]*string // Maps "ProviderID:Key" to pointer to value
	selectedSlice    []config.AppID
	lastProvider     string
}

func newSetupForm(ctx *wizardContext, initialRaw string) *setupForm {
	sf := &setupForm{
		ctx:              ctx,
		selectedProvider: ctx.providerID,
		lastProvider:     ctx.providerID,
		credentialValues: make(map[string]*string),
	}

	// Initialize selectedSlice from context
	for id, isSelected := range ctx.selected {
		if isSelected {
			sf.selectedSlice = append(sf.selectedSlice, id)
		}
	}

	// Pre-allocate credential storage for all providers
	for _, prov := range ctx.registry.All() {
		provID := prov.ID()
		for _, spec := range prov.RequiredCredentials() {
			val := new(string)
			sf.credentialValues[provID+":"+spec.Key] = val
			if provID == "exa" && spec.Key == "EXA_API_KEY" && len(ctx.profiles) == 0 {
				*val = initialRaw
			}
		}
	}

	sf.rebuildForm()
	return sf
}

func (sf *setupForm) rebuildForm() {
	appOptions := make([]huh.Option[config.AppID], len(sf.ctx.manager.Apps))
	for i, appConfig := range sf.ctx.manager.Apps {
		appOptions[i] = huh.NewOption(appConfig.Name, appConfig.ID)
	}

	allProviders := sf.ctx.registry.All()
	providerOptions := make([]huh.Option[string], len(allProviders))
	for i, prov := range allProviders {
		providerOptions[i] = huh.NewOption(prov.Name(), prov.ID())
	}

	fields := []huh.Field{
		huh.NewSelect[string]().
			Title("Provider").
			Description("Choose the MCP server profile to install").
			Options(providerOptions...).
			Value(&sf.selectedProvider),

		huh.NewMultiSelect[config.AppID]().
			Title("Target Apps").
			Description("Pick the local AI tools that should receive this MCP config").
			Options(appOptions...).
			Value(&sf.selectedSlice),
	}

	// Add fields for the SELECTED provider only
	prov, ok := sf.ctx.registry.Get(sf.selectedProvider)
	if ok && !sf.ctx.isPreloaded {
		for _, spec := range prov.RequiredCredentials() {
			val := sf.credentialValues[sf.selectedProvider+":"+spec.Key]
			var field huh.Field
			if spec.MultiValue {
				field = huh.NewText().
					Title(spec.Label).
					Description(spec.Description).
					Value(val).
					Validate(spec.Validator)
			} else {
				f := huh.NewInput().
					Title(spec.Label).
					Description(spec.Description).
					Value(val).
					Validate(spec.Validator)
				if spec.Secret {
					f.EchoMode(huh.EchoModePassword)
				}
				field = f
			}
			fields = append(fields, field)
		}
	}

	sf.form = huh.NewForm(huh.NewGroup(fields...)).WithTheme(huh.ThemeCatppuccin())
}

func (sf *setupForm) update(msg tea.Msg) (*setupForm, tea.Cmd) {
	_, cmd := sf.form.Update(msg)

	// If the provider has changed, we must rebuild the form to show the correct credential fields
	if sf.selectedProvider != sf.lastProvider {
		sf.lastProvider = sf.selectedProvider
		sf.rebuildForm()
		// After rebuilding, we need to Init the new form
		return sf, sf.form.Init()
	}

	return sf, cmd
}

func (sf *setupForm) syncToContext() {
	// Sync provider
	sf.ctx.providerID = sf.selectedProvider
	prov, _ := sf.ctx.registry.Get(sf.selectedProvider)
	sf.ctx.provider = prov

	// Sync selected apps
	newSelected := make(map[config.AppID]bool)
	for _, id := range sf.selectedSlice {
		newSelected[id] = true
	}
	sf.ctx.selected = newSelected

	// If profiles were NOT pre-loaded from flags, build them from form values
	if len(sf.ctx.profiles) == 0 && sf.ctx.provider != nil {
		specs := sf.ctx.provider.RequiredCredentials()
		profiles := []provider.CredentialProfile{}

		// In Phase 2, we assume a provider defines one set of credentials that might result in multiple profiles.
		// For Exa, the "EXA_API_KEY" multivalue field is parsed into multiple profiles.
		// For future single-value providers, it results in one profile.

		mv, isMultiValue := sf.ctx.provider.(provider.MultiValueParser)
		for _, spec := range specs {
			raw := *sf.credentialValues[sf.ctx.providerID+":"+spec.Key]
			if spec.MultiValue && isMultiValue {
				parsed, err := mv.ParseMultiValue(spec.Key, raw)
				if err == nil {
					profiles = append(profiles, parsed...)
				}
				continue
			}
			// Single-value credential: one profile with all creds gathered
		}
		// Build single-profile for non-multi-value providers
		if !isMultiValue || len(profiles) == 0 {
			values := make(map[string]string)
			label := "Default"
			for _, spec := range specs {
				val := *sf.credentialValues[sf.ctx.providerID+":"+spec.Key]
				values[spec.Key] = val
				if spec.Secret && label == "Default" && len(val) > 0 {
					label = redact.Key(val)
				}
			}
			profiles = append(profiles, provider.CredentialProfile{
				ProviderID: sf.ctx.providerID,
				Values:     values,
				Label:      label,
			})
		}
		sf.ctx.profiles = profiles
	}
}
