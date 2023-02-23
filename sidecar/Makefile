DOCKER_COMPOSE := docker compose -f docker/docker-compose.yaml

.env:
	cp .env.dist .env

.PHONY: docker-up
docker-up: .env
	$(DOCKER_COMPOSE) up -d

.PHONY: docker-down
docker-down:
	$(DOCKER_COMPOSE) down
