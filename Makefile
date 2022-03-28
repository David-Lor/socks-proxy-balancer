.DEFAULT_GOAL := help

IMAGE_NAME := "local/socks-proxy-balancer:latest"

build: ## docker build the image
	docker build . --pull -t "${IMAGE_NAME}"

test: ## run integration tests
	python tools/test.py

help: ## show this help.
	@fgrep -h "##" $(MAKEFILE_LIST) | fgrep -v fgrep | sed -e 's/\\$$//' | sed -e 's/##//'
