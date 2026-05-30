package uxexplore

import "github.com/nawodyaishan/universal-mcp-sync/pkg/tui"

func Fingerprint(m tui.DashboardModel) StateFingerprint {
	s := m.Snapshot()
	return StateFingerprint{
		Screen:            s.Screen,
		PreconditionClass: classifyPreconditionClass(s),
		BlockReason:       s.BlockReason,
		HasError:          s.HasScanError || s.HasPlanError || s.HasApplyError || s.HasValidationError,
		InFlight:          s.InFlight,
	}
}

func classifyPreconditionClass(s tui.DashboardSnapshot) string {
	switch {
	case s.HasScanError:
		return PCScanError
	case s.HasApplyError:
		return PCApplyError
	case s.HasPlanError:
		return PCPlanError
	case s.HasValidationError:
		return PCNetworkFailure
	case s.RuntimeMissing:
		return PCRuntimeMissing
	case s.ConflictUnresolved:
		return PCConflictUnresolved
	case s.MissingCredentials:
		return PCMissingCredentials
	case s.NoTargetsSelected:
		return PCNoTargetsSelected
	default:
		return PCOK
	}
}
