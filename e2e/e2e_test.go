//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hc-install/fs"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/terraform-exec/tfexec"
	"github.com/rogpeppe/go-internal/testscript"
)

// TerraformOutputs wraps tfexec output map with helper methods.
type TerraformOutputs map[string]tfexec.OutputMeta

// GetString returns the string value of an output.
func (o TerraformOutputs) GetString(key string) string {
	if meta, ok := o[key]; ok {
		var s string
		if err := json.Unmarshal(meta.Value, &s); err == nil {
			return s
		}
	}
	return ""
}

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{}))
}

func TestE2E(t *testing.T) {
	// Load Terraform outputs
	outputs, err := loadTerraformOutputs()
	if err != nil {
		t.Fatalf("Failed to load Terraform outputs: %v", err)
	}

	// Find ec2ssh binary
	ec2sshPath, err := findEC2SSHBinary()
	if err != nil {
		t.Fatalf("Failed to find ec2ssh binary: %v", err)
	}

	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Setup: func(env *testscript.Env) error {
			// Add ec2ssh binary to PATH
			binDir := filepath.Dir(ec2sshPath)
			env.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

			// Forward AWS credential environment variables
			for _, key := range []string{
				"AWS_PROFILE",
				"AWS_REGION",
				"AWS_DEFAULT_REGION",
				"AWS_ACCESS_KEY_ID",
				"AWS_SECRET_ACCESS_KEY",
				"AWS_SESSION_TOKEN",
				"AWS_SHARED_CREDENTIALS_FILE",
				"AWS_CONFIG_FILE",
				"HOME", // needed for ~/.aws/credentials
			} {
				if val := os.Getenv(key); val != "" {
					env.Setenv(key, val)
				}
			}

			// Set environment variables from Terraform outputs
			env.Setenv("EICE_ID", outputs.GetString("eice_id"))
			env.Setenv("PUBLIC_ID", outputs.GetString("public_id"))
			env.Setenv("PUBLIC_IP", outputs.GetString("public_ip"))
			env.Setenv("PRIVATE_ID", outputs.GetString("private_id"))
			env.Setenv("PRIVATE_IP", outputs.GetString("private_ip"))
			env.Setenv("PUBLIC_NAME", outputs.GetString("public_name"))
			env.Setenv("PRIVATE_NAME", outputs.GetString("private_name"))
			env.Setenv("EICE_IPV6_ID", outputs.GetString("eice_ipv6_id"))
			env.Setenv("IPV6_ONLY_ID", outputs.GetString("ipv6_only_id"))
			env.Setenv("IPV6_ONLY_IPV6", outputs.GetString("ipv6_only_ipv6"))
			env.Setenv("IPV6_ONLY_NAME", outputs.GetString("ipv6_only_name"))
			env.Setenv("USER", "ec2-user")

			return nil
		},
		ContinueOnError: true,
	})
}

// loadTerraformOutputs loads all outputs using terraform-exec.
func loadTerraformOutputs() (TerraformOutputs, error) {
	ctx := context.Background()

	terraformDir := filepath.Join(".", "terraform")
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		terraformDir = filepath.Join("..", "e2e", "terraform")
	}

	absDir, err := filepath.Abs(terraformDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Find terraform executable using hc-install
	finder := &fs.AnyVersion{Product: &product.Terraform}
	terraformPath, err := finder.Find(ctx)
	if err != nil {
		return nil, fmt.Errorf("terraform not found: %w", err)
	}

	tf, err := tfexec.NewTerraform(absDir, terraformPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create terraform instance: %w", err)
	}

	outputs, err := tf.Output(ctx)
	if err != nil {
		return nil, fmt.Errorf("terraform output failed (run 'make e2e-apply' first): %w", err)
	}

	return TerraformOutputs(outputs), nil
}

// findEC2SSHBinary locates the ec2ssh binary.
func findEC2SSHBinary() (string, error) {
	if path := os.Getenv("EC2SSH_BINARY"); path != "" {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	candidates := []string{
		filepath.Join(".", "ec2ssh"),
		filepath.Join("..", "e2e", "ec2ssh"),
		filepath.Join("..", "ec2ssh"),
	}

	for _, path := range candidates {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath, nil
			}
		}
	}

	return "", fmt.Errorf("ec2ssh binary not found (run 'make e2e-build' first)")
}
