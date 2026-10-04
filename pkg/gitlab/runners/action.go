package runners

import "strings"

// Action is a runners gitlab command action.
type Action string

const (
	ActionInspect Action = "inspect"
	ActionStop    Action = "stop"
	ActionStart   Action = "start"
	ActionRepair  Action = "repair"
	ActionRemove  Action = "remove"
)

// ParseAction normalizes a CLI action flag.
func ParseAction(s string) (Action, bool) {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "":
		return "", false
	case "inspect":
		return ActionInspect, true
	case "stop":
		return ActionStop, true
	case "start":
		return ActionStart, true
	case "repair":
		return ActionRepair, true
	case "remove":
		return ActionRemove, true
	default:
		return "", false
	}
}
