package ec2client

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2instanceconnect"
)

// SendSSHPublicKey pushes an SSH public key to an instance via EC2 Instance Connect.
func (c *Client) SendSSHPublicKey(instance types.Instance, instanceOSUser string, sshPublicKey string) error {
	c.logger.Printf("sending SSH public key to instance %s", *instance.InstanceId)

	input := &ec2instanceconnect.SendSSHPublicKeyInput{
		InstanceId:     aws.String(*instance.InstanceId),
		InstanceOSUser: aws.String(instanceOSUser),
		SSHPublicKey:   aws.String(sshPublicKey),
	}

	_, err := c.connectClient.SendSSHPublicKey(context.TODO(), input)
	if err == nil {
		c.logger.Printf("successfully sent SSH public key to instance %s", *instance.InstanceId)
	}

	return err
}
