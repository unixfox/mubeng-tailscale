package runner

import (
	"errors"

	"github.com/mubeng/mubeng/common"
	"github.com/mubeng/mubeng/internal/checker"
	"github.com/mubeng/mubeng/internal/daemon"
	"github.com/mubeng/mubeng/internal/server"
)

// New to switch an action, whether to check or run a proxy server.
func New(opt *common.Options) error {
	// Note: Tailscale tsnet manager is now initialized in options.go before proxy validation

	if opt.Address != "" {
		if opt.Daemon {
			return daemon.New(opt)
		}

		server.Run(opt)
	} else if opt.Check {
		checker.Do(opt)

		if opt.Output != "" {
			defer opt.Result.Close()
		}
	} else {
		return errors.New("no action to run")
	}

	return nil
}
