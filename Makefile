.PHONY: env start resume stop restart status vault-status

env:
	@./scripts/lifecycle/env.sh

start:
	@./scripts/lifecycle/start.sh

resume:
	@./scripts/lifecycle/resume.sh

stop:
	@./scripts/lifecycle/stop.sh

restart:
	@./scripts/lifecycle/restart.sh

status:
	@./scripts/lifecycle/status.sh

vault-status:
	@./scripts/lifecycle/vault-status.sh
