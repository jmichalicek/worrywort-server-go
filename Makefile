
worrywortd: worrywortd-gen
	go build ./cmd/worrywortd

worrywortd-gen:
	go generate ./...

# //bin/bash is windows msys make hack
dev:
	docker-compose run --service-ports --rm worrywortd //bin/bash

migrate-up:
	migrate -source file://./_migrations -database postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@${DATABASE_HOST}:5432/${DATABASE_NAME}?sslmode=disable up ${migrate_to}

migrate-down:
	migrate -source file://./_migrations -database postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@${DATABASE_HOST}:5432/${DATABASE_NAME}?sslmode=disable down ${migrate_to}

migrate-force:
	migrate -source file://./_migrations -database postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@${DATABASE_HOST}:5432/${DATABASE_NAME}?sslmode=disable force ${migrate_to}

seed-dev:
	psql -U ${DATABASE_USER} -h ${DATABASE_HOST} ${DATABASE_NAME} < _dev_seeds/seed.sql
	#migrate -source file://./_dev_seeds -database postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@${DATABASE_HOST}:5432/${DATABASE_NAME}?sslmode=disable up ${migrate_to}

psql:
	psql -U ${DATABASE_USER} -h ${DATABASE_HOST} ${DATABASE_NAME}

setup-test-db:
	dropdb -h ${DATABASE_HOST} -U ${DATABASE_USER} --if-exists ${DATABASE_NAME}_test
	createdb -h ${DATABASE_HOST} -U ${DATABASE_USER} ${DATABASE_NAME}_test -O ${DATABASE_USER}
	migrate -source file://./_migrations -database postgres://${DATABASE_USER}:${DATABASE_PASSWORD}@${DATABASE_HOST}:5432/${DATABASE_NAME}_test?sslmode=disable up ${migrate_to}

testcover:
	go test ${module} -cover -coverprofile=coverage.out

showcover:
	go tool cover -func=coverage.out

codeship-test: setup-test-db
	bash ./codecovtest.sh

codecov-upload:
	curl -s https://codecov.io/bash | bash

gofmt:
	gofmt -w ./
