//go:build e2e_live && (darwin || windows)

package fixture

import (
	"github.com/behaviorengineering/runnerconcierge/pkg/errdefs"
	"github.com/behaviorengineering/runnerconcierge/pkg/inventory"
)

func hasExpectedBlockingForFixture(rep *inventory.Report, configPath, serviceName string) bool {
	if !hasBlockingForFixture(rep, configPath, serviceName) {
		return false
	}
	want := expectedBlockingCodes()
	for _, code := range want {
		if hasFixtureFindingCode(rep, configPath, serviceName, code) {
			return true
		}
	}
	return false
}

func hasBlockingForFixture(rep *inventory.Report, configPath, serviceName string) bool {
	for _, f := range fixtureFindings(rep, configPath, serviceName) {
		if f.Block {
			return true
		}
	}
	return false
}

func hasFixtureFindingCode(rep *inventory.Report, configPath, serviceName string, code errdefs.Code) bool {
	for _, f := range fixtureFindings(rep, configPath, serviceName) {
		if f.Block && f.Code == code {
			return true
		}
	}
	return false
}

func fixtureFindings(rep *inventory.Report, configPath, serviceName string) []inventory.Finding {
	if rep == nil {
		return nil
	}
	var out []inventory.Finding
	for _, f := range rep.Findings {
		if f.ConfigPath == configPath {
			out = append(out, f)
			continue
		}
		if serviceName != "" && f.Service == serviceName {
			out = append(out, f)
		}
	}
	return out
}
