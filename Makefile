
docker-dev:
	docker-compose run --service-ports --rm worrywortd /bin/bash

migrate-up:
	migrate -source file://./_migrations -database postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:5432/${DB_NAME}?sslmode=disable up ${migrate_to}

migrate-down:
	migrate -source file://./_migrations -database postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:5432/${DB_NAME}?sslmode=disable down ${migrate_to}

migrate-force:
	migrate -source file://./_migrations -database postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:5432/${DB_NAME}?sslmode=disable force ${migrate_to}

# This is needed for hydra to work initially
hydra-migrate:
	docker-compose run hydra migrate sql postgres://developer:developer@database:5432/hydra?sslmode=disable
