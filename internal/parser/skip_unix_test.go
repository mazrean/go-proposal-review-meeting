//go:build unix

package parser_test

import (
	"os"
	"testing"
)

// skipIfNoPermissionEnforcement skips the test if file permissions are not enforced
// (e.g., when running as root on Unix systems).
func skipIfNoPermissionEnforcement(t *testing.T) {
	t.Helper()
	if os.Getuid() == 0 {
		t.Skip("skipping permission test when running as root")
	}
}
