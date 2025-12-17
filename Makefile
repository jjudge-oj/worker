all:
	echo "what"

.PHONY: prepare-dirs
prepare-dirs:
	@sudo mkdir -p /tmp/castletown/storage
	@sudo mkdir -p /tmp/castletown/libcontainer
	@sudo mkdir -p /tmp/castletown/overlayfs
	@sudo mkdir -p /tmp/castletown/rootfs
	@sudo mkdir -p /tmp/castletown/work
	@sudo mkdir -p /var/castletown/images
	@sudo mkdir -p /var/castletown/testcases

.PHONY: make-rootfs
make-rootfs: prepare-dirs
	sudo bash scripts/rootfs.sh

.PHONY: build
build:
	bash scripts/build.sh

.PHONY: dev
dev:
	sudo env "PATH=$$PATH:/usr/local/go/bin" go run main.go start

.PHONY: docker-up
docker-up:
	docker compose up --build -d

.PHONY: docker-down
docker-down:
	docker compose down

.PHONY: docker-logs
docker-logs:
	docker compose logs -f

.PHONY: migrate-up
migrate-up:
	migrate -path db/migrations -database "postgres://castletown:castletown@localhost:5432/castletown?sslmode=disable" up

.PHONY: migrate-down
migrate-down:
	migrate -path db/migrations -database "postgres://castletown:castletown@localhost:5432/castletown?sslmode=disable" down

.PHONY: migrate-create
migrate-create:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir db/migrations -seq $$name


.PHONY: test-grader
test-grader:
	@echo "Running grader tests..."
	sudo env "PATH=$$PATH:/usr/local/go/bin" go test github.com/joshjms/castletown/internal/grader -v

.PHONY: test-worker
test-worker: make-rootfs
	@echo "Running worker tests..."
	sudo env "PATH=$$PATH:/usr/local/go/bin" go test github.com/joshjms/castletown/internal/worker -v

.PHONY: test-container
test-container: make-rootfs
	@echo "Running container tests..."
	sudo env "PATH=$$PATH:/usr/local/go/bin" go test github.com/joshjms/castletown/internal/container -v
