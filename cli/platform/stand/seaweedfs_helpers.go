package stand

import (
	"strings"

	"github.com/MonadIAM/infrastructure/cli/platform/command"
)

func seaweedFSResponse(request command.Request) command.Result {
	arguments := strings.Join(request.Args, " ")
	if strings.Contains(arguments, "get vaultstaticsecrets.") {
		return command.Result{Output: []byte(`{"metadata":{"generation":1},"status":{"lastGeneration":1,"secretMAC":"ready","conditions":[{"type":"Ready","status":"True"},{"type":"Healthy","status":"True"}]}}`)}
	}

	if strings.Contains(arguments, "image inspect") {
		return command.Result{Output: []byte("sha256:" + strings.Repeat("a", 64))}
	}

	return command.Result{Output: []byte("kind: ConfigMap\ndata:\n  retention: ${SEAWEEDFS_AUDIT_LOG_RETENTION_DAYS}\n")}
}
