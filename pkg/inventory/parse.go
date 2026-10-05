package inventory

import (
	"errors"
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type tomlConfigError struct {
	err        error
	diagnostic string
}

func (e *tomlConfigError) Error() string {
	return e.err.Error()
}

func (e *tomlConfigError) Unwrap() error {
	return e.err
}

func parseConfigFile(path string) ([]RunnerEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config map[string]any
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, &tomlConfigError{err: err, diagnostic: "TOML decode failed"}
	}
	rawRunners, ok := config["runners"]
	if !ok {
		return nil, nil
	}
	runners, ok := rawRunners.([]any)
	if !ok {
		return nil, &tomlConfigError{err: fmt.Errorf("runners is not an array of tables"), diagnostic: "runners field is not an array of tables"}
	}
	out := make([]RunnerEntry, 0, len(runners))
	for index, rawRunner := range runners {
		runner, ok := rawRunner.(map[string]any)
		if !ok {
			return nil, &tomlConfigError{err: fmt.Errorf("runner entry %d is not a table", index), diagnostic: "runner entry is not a table"}
		}
		out = append(out, RunnerEntry{
			Name:     stringValue(runner["name"]),
			URL:      stringValue(runner["url"]),
			Executor: stringValue(runner["executor"]),
			GitLabID: integerValue(runner["id"]),
		})
	}
	return out, nil
}

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func integerValue(value any) int {
	switch number := value.(type) {
	case int64:
		return int(number)
	case int:
		return number
	default:
		return 0
	}
}

func configDiagnostic(err error) string {
	var decodeFailure *tomlConfigError
	if !errors.As(err, &decodeFailure) {
		return "config file read failed"
	}
	var decodeErr *toml.DecodeError
	if errors.As(decodeFailure, &decodeErr) {
		row, column := decodeErr.Position()
		if row > 0 && column > 0 {
			return fmt.Sprintf("TOML decode failed at line %d, column %d", row, column)
		}
		return decodeFailure.diagnostic
	}
	return decodeFailure.diagnostic
}
