.PHONY: build
build:
	@go mod tidy
	@ginkgo --vv --flake-attempts=10 --timeout=10m intigration/
	@ginkgo -r --vv --flake-attempts=10 --timeout=30m 

.PHONY: test
test:
	@ginkgo -r