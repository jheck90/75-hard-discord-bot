.PHONY: build push run stop clean help

# Docker image configuration
IMAGE_NAME := jheck90/75-half-chub-bot
VERSION := latest

# Build the Docker image
build:
	@echo "Building Docker image..."
	docker build -t $(IMAGE_NAME):$(VERSION) .
	@echo "✅ Build complete: $(IMAGE_NAME):$(VERSION)"

# Tag the image with version and latest
tag:
	@echo "Tagging image..."
	docker tag $(IMAGE_NAME):$(VERSION) $(IMAGE_NAME):latest
	@echo "✅ Tagged: $(IMAGE_NAME):latest"

# Push image to Docker Hub
push: build tag
	@echo "Pushing image to Docker Hub..."
	docker push $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME):latest
	@echo "✅ Pushed to Docker Hub: $(IMAGE_NAME)"

# Build and push in one command
build-push: push

# Run the container locally (requires .env file or environment variables)
run:
	@echo "Running container..."
	docker run -d \
		--name 75-half-chub-bot \
		--env-file .env \
		$(IMAGE_NAME):$(VERSION)

# Stop and remove the container
stop:
	@echo "Stopping container..."
	docker stop 75-half-chub-bot || true
	docker rm 75-half-chub-bot || true
	@echo "✅ Container stopped and removed"

# Clean up Docker images
clean:
	@echo "Cleaning up Docker images..."
	docker rmi $(IMAGE_NAME):$(VERSION) || true
	docker rmi $(IMAGE_NAME):latest || true
	@echo "✅ Cleanup complete"

# Show help
help:
	@echo "Available commands:"
	@echo "  make build       - Build the Docker image"
	@echo "  make push        - Build and push image to Docker Hub"
	@echo "  make build-push  - Alias for push"
	@echo "  make run         - Run the container locally (requires .env file)"
	@echo "  make stop        - Stop and remove the container"
	@echo "  make clean       - Remove local Docker images"
	@echo "  make help        - Show this help message"
	@echo ""
	@echo "Image: $(IMAGE_NAME):$(VERSION)"
