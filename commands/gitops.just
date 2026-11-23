set quiet
set working-directory := '..'
set dotenv-filename := '.env.dev'

# Create or update the Argo CD Image Updater Secret from the local GitHub App credentials.
credentials:
    @set -eu; \
        test -n "$GITHUB_APP_ID"; \
        test -n "$GITHUB_APP_INSTALLATION_ID"; \
        test -n "$GITHUB_APP_PRIVATE_KEY_FILE"; \
        test -n "$KUBECONFIG"; \
        test -n "$KUBE_CONTEXT"; \
        kubectl --kubeconfig "$KUBECONFIG" --context "$KUBE_CONTEXT" \
            -n argocd create secret generic monadiam-image-updater-github \
            --from-literal=githubAppID="$GITHUB_APP_ID" \
            --from-literal=githubAppInstallationID="$GITHUB_APP_INSTALLATION_ID" \
            --from-file=githubAppPrivateKey="$GITHUB_APP_PRIVATE_KEY_FILE" \
            --dry-run=client -o yaml | \
        kubectl --kubeconfig "$KUBECONFIG" --context "$KUBE_CONTEXT" \
            apply --server-side --field-manager=gitops-credentials -f -

# Forward the Argo CD UI to local port 8443.
ui:
    kubectl --kubeconfig "$KUBECONFIG" --context "$KUBE_CONTEXT" \
        -n argocd port-forward --address=127.0.0.1 \
        service/argocd-server 8443:443
