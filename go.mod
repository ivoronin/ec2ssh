module github.com/ivoronin/ec2ssh

go 1.25.8

require (
	al.essio.dev/pkg/shellescape v1.6.0
	github.com/aws/aws-sdk-go-v2 v1.41.7
	github.com/aws/aws-sdk-go-v2/config v1.32.17
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.299.1
	github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect v1.32.22
	github.com/aws/aws-sdk-go-v2/service/ssm v1.68.6
	github.com/gorilla/websocket v1.5.3
	github.com/hashicorp/hc-install v0.9.5
	github.com/hashicorp/terraform-exec v0.25.2
	github.com/ivoronin/argsieve v0.0.2
	github.com/mmmorris1975/ssm-session-client v0.403.0
	github.com/rogpeppe/go-internal v1.14.1
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/aws/aws-sdk-go v1.55.8 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.16 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.23 // indirect
	github.com/aws/aws-sdk-go-v2/internal/v4a v1.4.24 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.9 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.23 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.17 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.21 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.42.1 // indirect
	github.com/aws/session-manager-plugin v0.0.0-20250205214155-b2b0bcd769d1 // indirect
	github.com/aws/smithy-go v1.25.1 // indirect
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/eiannone/keyboard v0.0.0-20220611211555-0d226195f203 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.9.0 // indirect
	github.com/hashicorp/terraform-json v0.27.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/twinj/uuid v0.0.0-20151029044442-89173bcdda19 // indirect
	github.com/xtaci/smux v1.5.35 // indirect
	github.com/zclconf/go-cty v1.18.1 // indirect
	golang.org/x/crypto v0.49.0 // indirect
	golang.org/x/mod v0.35.0 // indirect
	golang.org/x/net v0.52.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.42.0 // indirect
	golang.org/x/term v0.41.0 // indirect
	golang.org/x/text v0.35.0 // indirect
	golang.org/x/tools v0.43.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/mmmorris1975/ssm-session-client => github.com/ivoronin/ssm-session-client v0.0.0-20251210165256-7a67290e8efb
