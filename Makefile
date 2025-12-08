.PHONY: lint test build demo clean e2e-init e2e-plan e2e-apply e2e-destroy e2e-output e2e-build e2e-test e2e

lint:
	golangci-lint run

test:
	go test ./...  # e2e tests excluded (require -tags=e2e)

build:
	go build ./cmd/ec2ssh

demo:
	cd demo && vhs demo.vhs

clean:
	rm -rf dist/

# E2E Testing Infrastructure
E2E_TF_DIR := e2e/terraform

e2e-init:
	cd $(E2E_TF_DIR) && terraform init

e2e-plan:
	cd $(E2E_TF_DIR) && terraform plan

e2e-apply:
	cd $(E2E_TF_DIR) && terraform apply

e2e-destroy:
	cd $(E2E_TF_DIR) && terraform destroy

e2e-output:
	cd $(E2E_TF_DIR) && terraform output

# E2E Testing
e2e-build:
	go build -o e2e/ec2ssh ./cmd/ec2ssh

e2e-test: e2e-build
	cd e2e && go test -v -tags=e2e ./...

# Full E2E workflow (assumes infrastructure is already up)
e2e: e2e-build e2e-test
