package uxexplore

// IsTerminal reports whether a fingerprint represents a state from which the
// probe should stop expanding edges. Terminal states are cases where remaining
// in the same (Screen, PreconditionClass) is the intended product behavior, not
// a dead-end.
//
// Justifications:
//   - ApplyResult is the success/failure landing after an apply; the user
//     leaves via [q] quit, which exits the program rather than progressing the
//     state machine. Treating it as terminal prevents the dead-end detector
//     from flagging it.
//   - Doctor with scan-error after a rescan attempt is the only sane resting
//     state when scanning cannot succeed: the model has shown the error, the
//     user has retried, and continuing to fire keys would re-enter the same
//     scan-error edge.
func IsTerminal(fp StateFingerprint) bool {
	switch fp.Screen {
	case "ApplyResult":
		return true
	case "Doctor":
		// Treat the persistent scan-error state as terminal — the analyzer
		// inspects the trace to confirm a rescan was attempted (PR 14d).
		return fp.PreconditionClass == PCScanError
	}
	return false
}
