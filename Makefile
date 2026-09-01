.PHONY: up down backend build dockerfile minikube-up minikube-down kpf preview kpf-stop pf-stop

up:
	set -a; . ./.env; set +a; docker compose up -d
	$(MAKE) kpf
	$(MAKE) preview

down:
	docker compose down -v
	-$(MAKE) pf-stop

minikube-up:
	minikube start --driver=docker --cpus=4 --memory=8g

minikube-down:
	minikube stop

backend:
	air

frontend:
	cd web && npm run dev

dockerfile:
	docker build -t visualkb-server:dev .

build: dockerfile

# In-cluster proxy: controller-exec (backend uses for EnsureRunning / GetStatus / Exec).
kpf:
	@if lsof -nP -iTCP:8082 -sTCP:LISTEN >/dev/null 2>&1; then \
		echo "8082 already forwarded"; \
	else \
		nohup kubectl -n agent-sandbox-system port-forward \
			svc/controller-exec 8082:8082 --address 127.0.0.1 \
			> /tmp/kpf-controller.log 2>&1 & \
	fi

# In-cluster proxy: static-router (iframe hits *.preview.example.com via host:8080).
# Pair with `*.preview.example.com 127.0.0.1` in /etc/hosts.
preview:
	@if lsof -nP -iTCP:8080 -sTCP:LISTEN >/dev/null 2>&1; then \
		echo "8080 already forwarded"; \
	else \
		nohup kubectl -n agent-sandbox-system port-forward \
			svc/static-router 8080:8080 --address 127.0.0.1 \
			> /tmp/kpf-router.log 2>&1 & \
	fi

pf-stop:
	-@pkill -f "kubectl.*port-forward.*controller-exec" 2>/dev/null || true
	-@pkill -f "kubectl.*port-forward.*static-router" 2>/dev/null || true