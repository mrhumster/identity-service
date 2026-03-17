IMAGE_NAME := xomrkob/identity-service
NAMESPACE := go-app
DEPLOYMENT := identity-service
VERSION ?= $(shell git describe --tags --always || echo "latest")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

.PHONY: all build push deploy clean

all: build push deploy

build:
	@echo "Building docker image $(IMAGE_NAME):$(VERSION)..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_DATE=$(BUILD_DATE) \
		-t $(IMAGE_NAME):$(VERSION) \
		-t $(IMAGE_NAME):latest .

push:
	@echo "Pushing image $(IMAGE_NAME):$(VERSION)..."
	docker push $(IMAGE_NAME):$(VERSION)
	docker push $(IMAGE_NAME):latest

deploy:
	@echo "Updating K8s deployment..."
	kubectl -n $(NAMESPACE) set image deployment/$(DEPLOYMENT) \
		identity-service=$(IMAGE_NAME):$(VERSION)
	@echo "Success!"

test:
	go test -v ./...

logs:
	kubectl -n $(NAMESPACE) logs -f -l app=identity-service

deploy-postgres:
	helm -n go-app install postgresql oci://registry-1.docker.io/bitnamicharts/postgresql -f ~/projects/identity-service/deploy/k8s/base/values.yaml

deploy-redis:
	helm install casbin-redis oci://registry-1.docker.io/bitnamicharts/redis --namespace go-app --set architecture=standalone --set auth.enabled=true --set auth.password=password --set master.persistence.enabled=false

deploy-certmanager:
	kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.19.4/cert-manager.yaml

deploy-ingress-nginx:
	kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.2/deploy/static/provider/cloud/deploy.yaml
