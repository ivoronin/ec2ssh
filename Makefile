DISABLED_LINTERS=nonamedreturns,wrapcheck,gomnd,nestif,exhaustruct,exhaustivestruct,depguard
EXCLUDED_ISSUES=(tainted input or cmd|Function 'Test.+' has too many statements|aws.+ is a global variable|DstTypeNames is a global variable)

.PHONY: test lint

all: test lint

lint:
	golangci-lint run --enable-all --disable ${DISABLED_LINTERS} -e "${EXCLUDED_ISSUES}"

test:
	go test ./...