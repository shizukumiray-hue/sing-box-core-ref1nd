//go:build !android || disable_pidfd_workaround

package libbox

import (
	"os"
	_ "unsafe"
)

// https://github.com/SagerNet/sing-box/issues/3233
// https://github.com/golang/go/issues/70508
// https://github.com/tailscale/tailscale/issues/13452
// 
// NOTE: This workaround is disabled for gomobile builds because
// go:linkname cannot be used in c-shared buildmode.
// The original issue should not affect gomobile-built libraries.

//go:linkname checkPidfdOnce os.checkPidfdOnce
var checkPidfdOnce func() error

func init() {
	checkPidfdOnce = func() error {
		return os.ErrInvalid
	}
}
