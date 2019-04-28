# not a fan of mixed stuff to run IN the dev container and stuff to run outside of it, but whatever
PRODUCTION_REPO	:= worrywort/worrywort-server
COMMIT_SHA	:= $$(git rev-parse --short HEAD)
BRANCH_NAME := $$(git branch | grep \* | cut -d ' ' -f2)
PRODUCTION_IMG	:= ${PRODUCTION_REPO}:${BRANCH_NAME}_${COMMIT_SHA}

# ugh. none of this works from windows
# Production docker image build
docker-image:
	docker build -t ${PRODUCTION_IMG} .

production-image:
	docker build -t ${PRODUCTION_IMG} .
	docker tag ${PRODUCTION_IMG} ${PRODUCTION_REPO}:latest

push-prod-image:
	docker push ${PRODUCTION_IMG}
	docker push ${PRODUCTION_REPO}:latest

# End production docker image

# Commands for outside of docker image/container
# //bin/bash is windows msys make hack
dev:
	docker-compose run --service-ports --rm worrywortd //bin/bash
# end outside of dev container

# Development tools in dev image
worrywortd: worrywortd-gen
	go build ./cmd/worrywortd

worrywortd-gen:
	go generate ./...

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
