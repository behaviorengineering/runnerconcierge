//go:build e2e_live && darwin

package fixture

import "context"

func runSudoUninstall(ctx context.Context, bin, serviceName string) error {
	_ = runSudo(ctx, bin, "stop", "--service", serviceName)
	return runSudo(ctx, bin, "uninstall", "--service", serviceName)
}
