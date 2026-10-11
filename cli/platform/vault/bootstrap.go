package vault

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)

func (client Client) ensureMount(ctx context.Context, kind, name string, enable ...string) error {
	result, err := client.authenticated(ctx, nil, kind, "list", "-format=json")
	if err != nil {
		return err
	}

	var mounts map[string]json.RawMessage
	err = json.Unmarshal(result.Output, &mounts)
	if err != nil {
		return fmt.Errorf("invalid Vault %s list", kind)
	}

	_, exists := mounts[name+"/"]
	if exists {
		return nil
	}

	_, err = client.authenticated(ctx, nil, append([]string{kind, "enable"}, enable...)...)

	return err
}

func (client Client) Bootstrap(ctx context.Context) error {
	env := client.Environment

	err := client.ensureMount(ctx, "secrets", env["VAULT_KV_MOUNT"], "-path="+env["VAULT_KV_MOUNT"], "kv-v2")
	if err != nil {
		return err
	}

	err = client.ensureMount(ctx, "auth", env["VAULT_K8S_AUTH_MOUNT"], "-path="+env["VAULT_K8S_AUTH_MOUNT"], "kubernetes")
	if err != nil {
		return err
	}

	_, err = client.authenticated(ctx, nil, "write", "auth/"+env["VAULT_K8S_AUTH_MOUNT"]+"/config", "kubernetes_host=https://kubernetes.default.svc:443")
	if err != nil {
		return err
	}

	for _, binding := range roleDefinitions() {
		err := client.configureRole(ctx, binding.Namespace, env[binding.PolicyKey], env[binding.RoleKey])
		if err != nil {
			return err
		}
	}

	err = client.ensureMount(ctx, "audit", "file", "file", "file_path=/vault/audit/audit.log")
	if err != nil {
		return err
	}

	err = client.EnsureCredentials(ctx)
	if err != nil {
		return err
	}

	namespaces := []string{"monadiam-apps", "monadiam-data", "monadiam-messaging", "monadiam-observability"}

	for _, namespace := range namespaces {
		err := client.put(ctx, namespace+"/bootstrap", map[string]string{"status": "configured"}, "")
		if err != nil {
			return err
		}
	}

	return nil
}

func (client Client) configureRole(ctx context.Context, namespace, policy, role string) error {
	data, err := os.ReadFile("platform/vault/bootstrap/" + namespace + "-read.hcl")
	if err != nil {
		return err
	}

	_, err = client.authenticated(ctx, data, "policy", "write", policy, "-")
	if err != nil {
		return err
	}

	env := client.Environment
	_, err = client.authenticated(ctx, nil, "write", "auth/"+env["VAULT_K8S_AUTH_MOUNT"]+"/role/"+role,
		"bound_service_account_names=vault-auth", "bound_service_account_namespaces="+namespace,
		"policies="+policy, "audience="+env["VAULT_K8S_AUDIENCE"], "ttl="+env["VAULT_K8S_TTL"])

	return err
}
