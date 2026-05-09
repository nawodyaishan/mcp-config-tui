package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/nawodyaishan/mcp-config-tui/pkg/config"
	"github.com/nawodyaishan/mcp-config-tui/pkg/exa"
	"github.com/nawodyaishan/mcp-config-tui/pkg/provider"
)

type setupForm struct {
	form             *huh.Form
	ctx              *wizardContext
	selectedProvider string
	credentialValues map[string]*string // Maps "ProviderID:Key" to pointer to value
	selectedSlice    []config.AppID
}

func newSetupForm(ctx *wizardContext, initialRaw string) *setupForm {
	sf := &setupForm{
		ctx:              ctx,
		selectedProvider: ctx.providerID,
		credentialValues: make(map[string]*string),
	}

	// Initialize selectedSlice from context
	for id, isSelected := range ctx.selected {
		if isSelected {
			sf.selectedSlice = append(sf.selectedSlice, id)
		}
	}

	appOptions := make([]huh.Option[config.AppID], len(ctx.manager.Apps))
	for i, appConfig := range ctx.manager.Apps {
		appOptions[i] = huh.NewOption(appConfig.Name, appConfig.ID)
	}

	allProviders := ctx.registry.All()
	providerOptions := make([]huh.Option[string], len(allProviders))
	for i, prov := range allProviders {
		providerOptions[i] = huh.NewOption(prov.Name(), prov.ID())
	}

	groups := []*huh.Group{
		huh.NewGroup(
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
		),
	}

	// For each provider, add a group of credential fields
	// We use WithHideFunc to only show the relevant group
	for _, prov := range allProviders {
		provID := prov.ID()
		specs := prov.RequiredCredentials()
		fields := make([]huh.Field, 0, len(specs))

		for _, spec := range specs {
			// Allocate storage for this credential value
			val := new(string)
			sf.credentialValues[provID+":"+spec.Key] = val

			// Special case for backward compatibility: seed initialRaw into Exa key field if not pre-loaded
			if provID == "exa" && spec.Key == "EXA_API_KEY" && len(ctx.profiles) == 0 {
				*val = initialRaw
			}

			var field huh.Field
			if spec.MultiValue {
				// Use multiline text area for multivalue credentials (like Exa keys)
				field = huh.NewText().
					Title(spec.Label).
					Description(spec.Description).
					Value(val).
					Validate(spec.Validator)
			} else {
				// Use single line input, password if secret
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

		if len(fields) > 0 {
			group := huh.NewGroup(fields...).
				WithHideFunc(func() bool {
					return sf.selectedProvider != provID || len(ctx.profiles) > 0
				})
			groups = append(groups, group)
		}
	}

	sf.form = huh.NewForm(groups...).WithTheme(huh.ThemeCatppuccin())

	return sf
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

		// Let's handle Exa's multi-value parsing specifically for now, while allowing generic fallback.
		if sf.ctx.providerID == "exa" {
			rawKeys := *sf.credentialValues["exa:EXA_API_KEY"]
			keys, _ := exa.ParseKeys(rawKeys)
			for _, key := range keys {
				profiles = append(profiles, provider.CredentialProfile{
					ProviderID: "exa",
					Values:     map[string]string{"EXA_API_KEY": key},
					Label:      exa.RedactKey(key),
				})
			}
		} else {
			// Generic fallback: one profile with all gathered values for this provider
			values := make(map[string]string)
			for _, spec := range specs {
				values[spec.Key] = *sf.credentialValues[sf.ctx.providerID+":"+spec.Key]
			}
			profiles = append(profiles, provider.CredentialProfile{
				ProviderID: sf.ctx.providerID,
				Values:     values,
				Label:      "Default", // TODO: Better generic labelling
			})
		}
		sf.ctx.profiles = profiles
	}
}
