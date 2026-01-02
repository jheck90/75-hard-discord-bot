.PHONY: build push run stop clean help

IMAGE := jheck90/75-half-chub-bot
# VERSION := v1.0.0 (set to desired version)

build:
	docker build -t $(IMAGE):$(VERSION) -t $(IMAGE):latest .

push: build
	docker push $(IMAGE):$(VERSION) && docker push $(IMAGE):latest

run:
	docker run -d --name 75-half-chub-bot --env-file .env $(IMAGE):latest

stop:
	docker stop 75-half-chub-bot 2>/dev/null; docker rm 75-half-chub-bot 2>/dev/null

clean:
	docker rmi $(IMAGE):$(VERSION) $(IMAGE):latest 2>/dev/null

help:
	@echo "make build  - Build image ($(IMAGE):$(VERSION))"
	@echo "make push   - Build and push to Docker Hub"
	@echo "make run    - Run container (requires .env)"
	@echo "make stop   - Stop and remove container"
	@echo "make clean  - Remove local images"
	@echo ""
	@echo "Override version: make push VERSION=1.0.1"
