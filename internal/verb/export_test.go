package verb

import "github.com/23min/aiwf/internal/gitops"

// ValidateAuthorizeTrailersForTest exposes the package-private
// validateAuthorizeTrailers helper for unit-level testing from
// the external verb_test package (M-0161/AC-2 reviewer S-2
// follow-up — pins the verb→gitops.ValidateTrailer seam that the
// AC-2 rung-pair check no longer exercises via the integration
// path).
//
// Test-only: lives in _test.go so it never compiles into
// production binaries.
func ValidateAuthorizeTrailersForTest(trailers []gitops.Trailer) error {
	return validateAuthorizeTrailers(trailers)
}
