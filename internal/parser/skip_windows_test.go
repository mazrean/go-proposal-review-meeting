//go:build windows

package parser_test

import "testing"

// skipIfNoPermissionEnforcement skips the test on Windows since file permissions
// are not enforced the same way as on Unix systems.
func skipIfNoPermissionEnforcement(t *testing.T) {
	t.Helper()
	t.Skip("skipping permission test on Windows")
}
