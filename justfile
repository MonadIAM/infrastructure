set quiet
set default-list

mod env 'commands/env.just'
mod cli 'commands/cli.just'
mod host 'commands/host.just'
mod k8 'commands/k8.just'
mod zot 'commands/zot.just'
mod tf 'commands/tf.just'
mod platform 'commands/platform.just'
mod vault 'commands/vault.just'
mod gitops 'commands/gitops.just'
mod app 'commands/app.just'

# Start the prepared stand and verify all five API documentation endpoints.
start:
    just env::init
    just k8::start
    just zot::apply
    just tf::apply storage
    just platform::reconcile cert-manager
    just platform::reconcile metrics-server
    just platform::reconcile istio
    just platform::ambient
    just platform::reconcile vault
    just platform::reconcile gateway
    just platform::reconcile operators
    just platform::reconcile clusters
    just platform::reconcile apicurio
    just platform::reconcile seaweedfs
    just tf::apply observability
    just tf::apply gitops
    just tf::apply gitops-apps
    just platform::observability-ready
    just app::ready

# Reconcile and start the complete stand after it was stopped.
resume: start

# Stop MicroK8s and all stand components running inside it.
stop:
    just k8::stop

# Stop and start the stand, then verify the five API endpoints.
restart:
    just stop
    just start

# Show a summary of platform health through the Go CLI.
status:
    just platform::status
