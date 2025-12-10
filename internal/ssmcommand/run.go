// Package ssmcommand provides SSM RunCommand execution.
package ssmcommand

import (
	"context"
	"errors"
	"fmt"
	"time"

	"al.essio.dev/pkg/shellescape"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// RunCommand executes a command on an EC2 instance via SSM RunCommand API.
// Returns stdout, stderr content and error.
// On non-zero exit code, returns *ExitError which implements ExitCode() int.
// Arguments are properly shell-quoted before execution.
func RunCommand(ctx context.Context, cfg aws.Config, instanceID string, args []string) (stdout, stderr string, err error) {
	client := ssm.NewFromConfig(cfg)

	// Shell-quote arguments to preserve spaces and special characters
	command := shellescape.QuoteCommand(args)

	// Send command using AWS-RunShellScript document
	sendOutput, err := client.SendCommand(ctx, &ssm.SendCommandInput{
		InstanceIds:  []string{instanceID},
		DocumentName: aws.String("AWS-RunShellScript"),
		Parameters:   map[string][]string{"commands": {command}},
	})
	if err != nil {
		return "", "", fmt.Errorf("failed to send command: %w", err)
	}

	if sendOutput.Command == nil || sendOutput.Command.CommandId == nil {
		return "", "", errors.New("SSM returned empty command response")
	}

	commandID := *sendOutput.Command.CommandId

	// Poll for completion
	return waitForCompletion(ctx, client, commandID, instanceID)
}

// waitForCompletion polls SSM until the command reaches a terminal state.
func waitForCompletion(ctx context.Context, client *ssm.Client, commandID, instanceID string) (stdout, stderr string, err error) {
	// Exponential backoff parameters
	interval := 100 * time.Millisecond
	maxInterval := 5 * time.Second

	for {
		select {
		case <-ctx.Done():
			return "", "", fmt.Errorf("timeout waiting for command completion: %w", ctx.Err())
		default:
		}

		output, err := client.GetCommandInvocation(ctx, &ssm.GetCommandInvocationInput{
			CommandId:  aws.String(commandID),
			InstanceId: aws.String(instanceID),
		})
		if err != nil {
			// InvocationDoesNotExist might happen briefly after SendCommand - retry
			var notFound *ssmtypes.InvocationDoesNotExist
			if errors.As(err, &notFound) {
				time.Sleep(interval)
				continue
			}
			return "", "", fmt.Errorf("failed to get command status: %w", err)
		}

		stdout = aws.ToString(output.StandardOutputContent)
		stderr = aws.ToString(output.StandardErrorContent)

		switch output.Status {
		case ssmtypes.CommandInvocationStatusPending,
			ssmtypes.CommandInvocationStatusInProgress,
			ssmtypes.CommandInvocationStatusDelayed:
			// Still running - wait and retry with backoff
			time.Sleep(interval)
			interval = min(interval*2, maxInterval)
			continue

		case ssmtypes.CommandInvocationStatusSuccess:
			return stdout, stderr, nil

		case ssmtypes.CommandInvocationStatusFailed:
			exitCode := int(output.ResponseCode)
			if exitCode == 0 {
				exitCode = 1 // Ensure non-zero for failed commands
			}
			return stdout, stderr, &ExitError{Code: exitCode}

		case ssmtypes.CommandInvocationStatusTimedOut:
			return stdout, stderr, errors.New("command timed out on remote instance")

		case ssmtypes.CommandInvocationStatusCancelled,
			ssmtypes.CommandInvocationStatusCancelling:
			return stdout, stderr, errors.New("command was cancelled")

		default:
			return stdout, stderr, fmt.Errorf("unexpected command status: %s", output.Status)
		}
	}
}

// ExitError represents a remote command that exited with a non-zero code.
// It implements ExitCode() to allow main.go to extract the exit code.
type ExitError struct {
	Code int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("remote command exited with code %d", e.Code)
}

// ExitCode returns the exit code of the remote command.
func (e *ExitError) ExitCode() int {
	return e.Code
}
