package prompt

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
)

// Prompter collects interactive answers.
type Prompter interface {
	Confirm(ctx context.Context, title string) (bool, error)
	Input(ctx context.Context, title, placeholder, value string) (string, error)
	Password(ctx context.Context, title string) (string, error)
	Select(ctx context.Context, title string, options []string) (int, error)
}

// Huh implements Prompter with charmbracelet/huh.
type Huh struct{}

// NewHuh returns an interactive prompter.
func NewHuh() *Huh {
	return &Huh{}
}

// Confirm asks a yes/no question.
func (h *Huh) Confirm(ctx context.Context, title string) (bool, error) {
	if h == nil {
		return false, fmt.Errorf("prompt: prompter is nil")
	}
	var ok bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().Title(title).Value(&ok),
		),
	)
	if err := form.Run(); err != nil {
		return false, err
	}
	return ok, nil
}

// Input asks for a single line.
func (h *Huh) Input(ctx context.Context, title, placeholder, value string) (string, error) {
	val := value
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(title).Placeholder(placeholder).Value(&val),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return val, nil
}

// Select asks the user to pick one option.
func (h *Huh) Select(ctx context.Context, title string, options []string) (int, error) {
	if len(options) == 0 {
		return 0, fmt.Errorf("prompt: no options")
	}
	var idx int
	huhOpts := make([]huh.Option[int], len(options))
	for i, label := range options {
		huhOpts[i] = huh.NewOption(label, i)
	}
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title(title).
				Options(huhOpts...).
				Value(&idx),
		),
	)
	if err := form.Run(); err != nil {
		return 0, err
	}
	return idx, nil
}

// Password asks for a secret line.
func (h *Huh) Password(ctx context.Context, title string) (string, error) {
	var val string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title(title).EchoMode(huh.EchoModePassword).Value(&val),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return val, nil
}

// NonInteractive fails closed on prompts.
type NonInteractive struct{}

func (n *NonInteractive) Confirm(ctx context.Context, title string) (bool, error) {
	return false, fmt.Errorf("prompt: non-interactive mode (%s)", title)
}

func (n *NonInteractive) Input(ctx context.Context, title, placeholder, value string) (string, error) {
	return "", fmt.Errorf("prompt: non-interactive mode (%s)", title)
}

func (n *NonInteractive) Password(ctx context.Context, title string) (string, error) {
	return "", fmt.Errorf("prompt: non-interactive mode (%s)", title)
}

func (n *NonInteractive) Select(ctx context.Context, title string, options []string) (int, error) {
	return 0, fmt.Errorf("prompt: non-interactive mode (%s)", title)
}

// ForTTY reports whether stdin is a terminal.
func ForTTY() Prompter {
	if fi, err := os.Stdin.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
		return NewHuh()
	}
	return &NonInteractive{}
}
