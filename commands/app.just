set quiet
set working-directory := '..'
set dotenv-filename := '.env.dev'

# Wait for five API deployments and check their public /docs/ endpoints.
ready:
    @set -eu; \
        for service in access-control identity notification organization hr; do \
            kubectl --kubeconfig "$KUBECONFIG" --context "$KUBE_CONTEXT" \
                -n monadiam-apps rollout status \
                "deployment/$service-service" --timeout=900s; \
        done
    @set -eu; \
        for port in 4001 4002 4003 4004 4005; do \
            curl --insecure --fail --silent --show-error --location \
                --retry-all-errors --retry 10 --retry-delay 2 \
                "https://$APPLICATIONS_IP:$port/docs/" --output /dev/null; \
        done
