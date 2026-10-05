package cleanup

import (
	"fmt"
	"io"
	"time"

	"github.com/behaviorengineering/gitvalet/pkg/gitexec"
)

// Config builds a cleanup controller.
type Config struct {
	Exec     gitexec.Exec
	Out      io.Writer
	ErrOut   io.Writer
	MinAge   time.Duration
	Interval time.Duration
	AllowYes bool
	SelfPath string
	Clock    func() time.Time
}

// Controller runs docker leftover cleanup and schedule install.
type Controller struct {
	cfg Config
}

// Create builds a Controller from Config.
func (cfg Config) Create() (*Controller, error) {
	if cfg.Exec == nil {
		panic("cleanup: exec is nil")
	}
	if cfg.Out == nil {
		return nil, fmt.Errorf("cleanup: Out writer is required")
	}
	if cfg.MinAge <= 0 {
		cfg.MinAge = time.Hour
	}
	if cfg.Interval <= 0 {
		cfg.Interval = 10 * time.Minute
	}
	if cfg.Clock == nil {
		cfg.Clock = time.Now
	}
	return &Controller{cfg: cfg}, nil
}
