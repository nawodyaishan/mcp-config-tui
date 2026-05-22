package app

import (
	"fmt"
	"strings"
	"time"
)

func FormatSavedPlan(plan SavedPlan, now time.Time) string {
	var builder strings.Builder
	builder.WriteString("Saved MCP plan\n")
	builder.WriteString("==============\n")
	fmt.Fprintf(&builder, "plan id: %s\n", plan.PlanID)
	fmt.Fprintf(&builder, "provider: %s\n", plan.ProviderID)
	fmt.Fprintf(&builder, "created: %s\n", plan.CreatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&builder, "expires: %s\n", plan.ExpiresAt.UTC().Format(time.RFC3339))
	if !plan.ExpiresAt.IsZero() && !now.IsZero() && plan.ExpiresAt.Before(now.UTC()) {
		builder.WriteString("warning: plan is expired\n")
	}
	for _, warning := range plan.Warnings {
		builder.WriteString("warning: " + warning + "\n")
	}
	builder.WriteString("Operations\n")
	for _, op := range plan.Operations {
		fmt.Fprintf(&builder, "- %s [%s]\n", op.TargetName, op.Action)
		if op.FilePath != "" {
			fmt.Fprintf(&builder, "  path: %s\n", op.FilePath)
		}
		if len(op.CLICommand) > 0 {
			fmt.Fprintf(&builder, "  command: %s\n", strings.Join(op.CLICommand, " "))
		}
		fmt.Fprintf(&builder, "  summary: %s\n", op.Redacted)
		for _, warning := range op.Warnings {
			fmt.Fprintf(&builder, "  warning: %s\n", warning)
		}
	}
	builder.WriteString("No config files were written.\n")
	return strings.TrimRight(builder.String(), "\n")
}

func FormatSavedPlanPreflight(preflight SavedPlanPreflight, now time.Time) string {
	var builder strings.Builder
	builder.WriteString("Saved MCP apply preview\n")
	builder.WriteString("=======================\n")
	fmt.Fprintf(&builder, "plan id: %s\n", preflight.PlanID)
	fmt.Fprintf(&builder, "provider: %s\n", preflight.ProviderID)
	fmt.Fprintf(&builder, "created: %s\n", preflight.CreatedAt.UTC().Format(time.RFC3339))
	fmt.Fprintf(&builder, "expires: %s\n", preflight.ExpiresAt.UTC().Format(time.RFC3339))
	if !preflight.ExpiresAt.IsZero() && !now.IsZero() && preflight.ExpiresAt.Before(now.UTC()) {
		builder.WriteString("warning: plan is expired\n")
	}
	for _, warning := range preflight.Warnings {
		builder.WriteString("warning: " + warning + "\n")
	}
	if len(preflight.ApprovalPrompts) > 0 {
		builder.WriteString("Approvals\n")
		for _, prompt := range preflight.ApprovalPrompts {
			fmt.Fprintf(&builder, "- %s\n", prompt.Message)
		}
	}
	builder.WriteString("Operations\n")
	for _, op := range preflight.Operations {
		fmt.Fprintf(&builder, "- %s [%s]\n", op.TargetName, op.Action)
		if op.FilePath != "" {
			fmt.Fprintf(&builder, "  path: %s\n", op.FilePath)
		}
		if len(op.CLICommand) > 0 {
			fmt.Fprintf(&builder, "  command: %s\n", strings.Join(op.CLICommand, " "))
		}
		fmt.Fprintf(&builder, "  summary: %s\n", op.Redacted)
	}
	builder.WriteString("No config files were written.\n")
	return strings.TrimRight(builder.String(), "\n")
}
