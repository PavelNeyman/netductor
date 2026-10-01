VERSION ?= 0.7.0-dev
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: netductor netductor-release
netductor:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/netductor ./cmd/netductor

netductor-release:
	mkdir -p dist
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/netductor-linux-amd64 ./cmd/netductor
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o dist/netductor-linux-arm64 ./cmd/netductor
	cd dist && sha256sum netductor-linux-* > SHA256SUMS-netductor


.PHONY: agent-release
agent-release:
	mkdir -p dist
	@for arch in amd64 arm64 arm mipsle riscv64; do \
	  extra=""; \
	  case $$arch in mipsle) extra="GOMIPS=softfloat";; arm) extra="GOARM=7";; esac; \
	  env $$extra CGO_ENABLED=0 GOOS=linux GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" \
	    -o dist/netductor-agent-linux-$$arch ./cmd/netductor-agent; \
	done
