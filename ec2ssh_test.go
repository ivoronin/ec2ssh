package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGuessAWSDestinationType(t *testing.T) {
	t.Parallel()

	assert.Equal(t, DstTypePrivateIP, GuessAWSDestinationType("192.168.1.1"))
	assert.Equal(t, DstTypePublicIP, GuessAWSDestinationType("1.1.1.1"))
	assert.Equal(t, DstTypeID, GuessAWSDestinationType("i-1234567890abcdef0"))
	assert.Equal(t, DstTypePrivateDNSName, GuessAWSDestinationType("ip-192-168-1-1"))
	assert.Equal(t, DstTypeNameTag, GuessAWSDestinationType("test"))
	assert.Equal(t, DstTypeIPv6, GuessAWSDestinationType("fec1::1"))
	assert.Equal(t, DstTypeNameTag, GuessAWSDestinationType("[fec1::1]"))
}
