
docker-dev:
	docker-compose run --service-ports --rm worrywortd /bin/bash

migrate-up:
	migrate -source file://./_migrations -database postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:5432/${DB_NAME}?sslmode=disable up ${migrate_to}

migrate-down:
	migrate -source file://./_migrations -database postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:5432/${DB_NAME}?sslmode=disable down ${migrate_to}

migrate-force:
	migrate -source file://./_migrations -database postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:5432/${DB_NAME}?sslmode=disable force ${migrate_to}
