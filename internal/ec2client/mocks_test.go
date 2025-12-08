package ec2client

import "github.com/aws/aws-sdk-go-v2/service/ec2/types"

// This file provides package-internal lowercase aliases for the exported mock types
// and test helpers defined in testing.go.
//
// The mocks and helpers are exported (PascalCase) in testing.go so they can be used
// by other packages' tests (e.g., internal/app/integration_test.go).
// This file provides lowercase aliases for use within this package's tests,
// following Go convention of unexported names for internal use.

// Mock type aliases (lowercase for internal use)
type mockEC2API = MockEC2API
type mockEC2InstanceConnectAPI = MockEC2InstanceConnectAPI
type mockHTTPRequestSigner = MockHTTPRequestSigner

// instanceOption is the functional option type for makeInstance
type instanceOption func(*types.Instance)

// Helper function aliases (lowercase for internal use)
var (
	makeInstance       = MakeInstance
	withPrivateIP      = WithPrivateIP
	withPublicIP       = WithPublicIP
	withIPv6           = WithIPv6
	withVPC            = WithVPC
	withSubnet         = WithSubnet
	withNameTag        = WithNameTag
	withTag            = WithTag
	makeReservation    = MakeReservation
	makeDescribeOutput = MakeDescribeOutput
	makeEICE           = MakeEICE
	makeEICEOutput     = MakeEICEOutput
	newTestClient      = NewTestClient
)

// addrTypePtr returns a pointer to the AddrType value.
func addrTypePtr(t AddrType) *AddrType {
	return &t
}

// dstTypePtr returns a pointer to the DstType value.
func dstTypePtr(t DstType) *DstType {
	return &t
}
