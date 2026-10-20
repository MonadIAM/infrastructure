package vault

import (
	"context"
	"os"
)

func (client Client) Application(ctx context.Context, name string) error {
	err := client.ensureMount(ctx, "secrets", "transit", "-path=transit", "transit")
	if err != nil {
		return err
	}

	keys := [][]string{
		{"transit/keys/" + name + "-audit-mask-hmac", "type=hmac", "key_size=32"},
		{"transit/keys/" + name + "-audit-log-sig", "type=ed25519"},
	}

	if name == "identity" {
		keys = append(keys, identityKeys...)
	}

	for _, key := range keys {
		_, err = client.authenticated(ctx, nil, append([]string{"write"}, key...)...)
		if err != nil {
			return err
		}
	}

	policy, err := os.ReadFile("applications/" + name + "/config/policy.hcl")
	if err != nil {
		return err
	}

	_, err = client.authenticated(ctx, policy, "policy", "write", name+"-service", "-")
	if err != nil {
		return err
	}

	_, err = client.authenticated(ctx, nil, "write", "auth/"+client.Environment["VAULT_K8S_AUTH_MOUNT"]+"/role/"+name+"-service",
		"bound_service_account_names="+name+"-service", "bound_service_account_namespaces=monadiam-apps",
		"policies="+name+"-service", "audience="+client.Environment["VAULT_K8S_AUDIENCE"], "ttl="+client.Environment["VAULT_K8S_TTL"])

	return err
}
