package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGuessAWSDestinationType(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DstTypePrivateIP, GuessDestinationType("192.168.1.1"))
	assert.Equal(t, DstTypePublicIP, GuessDestinationType("1.1.1.1"))
	assert.Equal(t, DstTypeID, GuessDestinationType("i-1234567890abcdef0"))
	assert.Equal(t, DstTypePrivateDNSName, GuessDestinationType("ip-192-168-1-1"))
	assert.Equal(t, DstTypePrivateDNSName, GuessDestinationType("ip-192-168-1-1.compute.internal"))
	assert.Equal(t, DstTypePrivateDNSName, GuessDestinationType("i-1234567890abcdef0.compute.internal"))
	assert.Equal(t, DstTypeNameTag, GuessDestinationType("test"))
	assert.Equal(t, DstTypeIPv6, GuessDestinationType("fec1::1"))
	assert.Equal(t, DstTypeNameTag, GuessDestinationType("[fec1::1]"))
}
