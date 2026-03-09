module github.com/ivoronin/ec2ssh

go 1.25.0

require (
	al.essio.dev/pkg/shellescape v1.6.0
	github.com/aws/aws-sdk-go-v2 v1.41.3
	github.com/aws/aws-sdk-go-v2/config v1.32.11
	github.com/aws/aws-sdk-go-v2/service/ec2 v1.294.0
	github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect v1.32.18
	github.com/aws/aws-sdk-go-v2/service/ssm v1.68.2
	github.com/gorilla/websocket v1.5.3
	github.com/hashicorp/hc-install v0.9.3
	github.com/hashicorp/terraform-exec v0.25.0
	github.com/ivoronin/argsieve v0.0.2
	github.com/mmmorris1975/ssm-session-client v0.403.0
	github.com/rogpeppe/go-internal v1.14.1
	github.com/stretchr/testify v1.11.1
)

require (
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/aws/aws-sdk-go v1.55.8 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.19.11 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.18.19 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.4.19 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.7.19 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.8.5 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding v1.13.6 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.13.19 // indirect
	github.com/aws/aws-sdk-go-v2/service/signin v1.0.7 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.30.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.35.16 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.41.8 // indirect
	github.com/aws/session-manager-plugin v0.0.0-20250205214155-b2b0bcd769d1 // indirect
	github.com/aws/smithy-go v1.24.2 // indirect
	github.com/cihub/seelog v0.0.0-20170130134532-f561c5e57575 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/eiannone/keyboard v0.0.0-20220611211555-0d226195f203 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/hashicorp/go-version v1.8.0 // indirect
	github.com/hashicorp/terraform-json v0.27.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/twinj/uuid v0.0.0-20151029044442-89173bcdda19 // indirect
	github.com/xtaci/smux v1.5.35 // indirect
	github.com/zclconf/go-cty v1.17.0 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/mod v0.33.0 // indirect
	golang.org/x/net v0.49.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/term v0.39.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	golang.org/x/tools v0.41.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/mmmorris1975/ssm-session-client => github.com/ivoronin/ssm-session-client v0.0.0-20251210165256-7a67290e8efb
