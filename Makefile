OK_COLOR=\033[32;01m
NO_COLOR=\033[0m

ENVVARS=IMAGE_SERVICE_URL=127.0.0.1:8112 STORAGE_URL=127.0.0.1:8111 STORAGE_ACCESS_KEY=5F9HBcm8TZpJmb8r STORAGE_SECRET_KEY=XF8wEgaMmsH2B5ne RESIZE_ON_UPLOAD=1 RESIZE_ON_DOWNLOAD=1

run:
	$(ENVVARS) go run controllers.go health.go main.go middlewares.go resize.go

test:
	$(ENVVARS) go test -cover

cov:
	$(ENVVARS) go test -coverprofile=coverage.out && go tool cover -html=coverage.out

doc:
	swagger generate spec -mo ./swagger.json && swagger serve ./swagger.json

docker-run-dev:
	docker-compose -f ./docker-compose.yml -f ./docker-compose.dev.yml up

docker-build:
	@echo "$(OK_COLOR)==> Building Docker image$(NO_COLOR)"
	docker build --no-cache=true -t teryaew/img2sto:$(VERSION) .

docker-push:
	@echo "$(OK_COLOR)==> Pushing Docker image v$(VERSION) $(NO_COLOR)"
	docker push teryaew/img2sto:$(VERSION)

.PHONY: run test cov docker-run-dev docker-build docker-push
