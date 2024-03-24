package main

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func TestBuildSSHArgs(t *testing.T) {
	t.Parallel()

	session := &Session{
		options: Options{
			Login: "login",
			Port:  "2222",
		},
		instance: types.Instance{
			InstanceId: aws.String("instance-id"),
		},
		destinationAddr: "192.168.0.1",
		proxyCommand:    "proxy-command",
		privateKeyPath:  "/path/to/private/key",
	}

	sshArgs := session.buildSSHArgs()
	assert.Equal(t, []string{
		"-oProxyCommand=proxy-command",
		"-llogin",
		"-p2222",
		"-i/path/to/private/key",
		"-oHostKeyAlias=instance-id",
		"192.168.0.1",
	}, sshArgs)
}
