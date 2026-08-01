package utils

// IsValidServiceName reports whether s is a legal Compose service name:
// letters, digits, hyphen and underscore only. Shared by every "make a
// service" flow (servicefieldsstep, addservicemodal) so the rule can never
// drift between them - see D7 in docs/plans/image-search.md for why this
// function is shared while the UI step it validates for is deliberately
// not.
func IsValidServiceName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return false
		}
	}
	return true
}
