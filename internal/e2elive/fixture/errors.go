//go:build e2e_live && (darwin || windows)

package fixture

import (
	"errors"

	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
)

// ErrSkipped means the fixture did not run (environment not ready); not a failure unless E2E_LIVE_REQUIRE=1.
var ErrSkipped = errors.New("e2e fixture skipped")

func wrap(op string, code errdefs.Code, msg string, err error) error {
	return errdefs.New(op, code, msg, err)
}
