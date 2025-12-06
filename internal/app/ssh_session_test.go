package app

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/stretchr/testify/assert"
)

func TestAppendOptArg(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		args   []string
		format string
		value  string
		want   []string
	}{
		{
			name:   "non-empty value appends formatted arg",
			args:   []string{"-a"},
			format: "-b%s",
			value:  "test",
			want:   []string{"-a", "-btest"},
		},
		{
			name:   "empty value does not append",
			args:   []string{"-a"},
			format: "-b%s",
			value:  "",
			want:   []string{"-a"},
		},
		{
			name:   "nil args creates new slice",
			args:   nil,
			format: "-p%s",
			value:  "2222",
			want:   []string{"-p2222"},
		},
		{
			name:   "format with equals sign",
			args:   []string{},
			format: "-oProxyCommand=%s",
			value:  "some command",
			want:   []string{"-oProxyCommand=some command"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := appendOptArg(tt.args, tt.format, tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBaseArgs(t *testing.T) {
	t.Parallel()

	instanceID := "i-0123456789abcdef0"

	tests := []struct {
		name    string
		session baseSSHSession
		want    []string
	}{
		{
			name: "minimal args with instance ID only",
			session: baseSSHSession{
				instance: types.Instance{
					InstanceId: aws.String(instanceID),
				},
			},
			want: []string{"-oHostKeyAlias=i-0123456789abcdef0"},
		},
		{
			name: "with identity file",
			session: baseSSHSession{
				instance: types.Instance{
					InstanceId: aws.String(instanceID),
				},
				privateKeyPath: "/path/to/key",
			},
			want: []string{"-i/path/to/key", "-oHostKeyAlias=i-0123456789abcdef0"},
		},
		{
			name: "with proxy command",
			session: baseSSHSession{
				instance: types.Instance{
					InstanceId: aws.String(instanceID),
				},
				proxyCommand: "ec2ssh --eice-tunnel",
			},
			want: []string{"-oProxyCommand=ec2ssh --eice-tunnel", "-oHostKeyAlias=i-0123456789abcdef0"},
		},
		{
			name: "with passthrough args",
			session: baseSSHSession{
				instance: types.Instance{
					InstanceId: aws.String(instanceID),
				},
				PassArgs: []string{"-o", "StrictHostKeyChecking=no", "-v"},
			},
			want: []string{"-oHostKeyAlias=i-0123456789abcdef0", "-o", "StrictHostKeyChecking=no", "-v"},
		},
		{
			name: "full configuration",
			session: baseSSHSession{
				instance: types.Instance{
					InstanceId: aws.String(instanceID),
				},
				proxyCommand:   "ec2ssh --eice-tunnel",
				privateKeyPath: "/tmp/key",
				PassArgs:       []string{"-t"},
			},
			want: []string{"-oProxyCommand=ec2ssh --eice-tunnel", "-i/tmp/key", "-oHostKeyAlias=i-0123456789abcdef0", "-t"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.session.baseArgs()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSSHSessionBuildArgs(t *testing.T) {
	t.Parallel()

	instanceID := "i-0123456789abcdef0"

	tests := []struct {
		name    string
		session SSHSession
		want    []string
	}{
		{
			name: "basic SSH args",
			session: SSHSession{
				baseSSHSession: baseSSHSession{
					instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					destinationAddr: "10.0.0.1",
					Login:           "ec2-user",
				},
			},
			want: []string{"-oHostKeyAlias=i-0123456789abcdef0", "-lec2-user", "10.0.0.1"},
		},
		{
			name: "with port",
			session: SSHSession{
				baseSSHSession: baseSSHSession{
					instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					destinationAddr: "10.0.0.1",
					Login:           "ubuntu",
					Port:            "2222",
				},
			},
			want: []string{"-oHostKeyAlias=i-0123456789abcdef0", "-lubuntu", "-p2222", "10.0.0.1"},
		},
		{
			name: "with command and args",
			session: SSHSession{
				baseSSHSession: baseSSHSession{
					instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					destinationAddr: "10.0.0.1",
					Login:           "root",
				},
				CommandWithArgs: []string{"ls", "-la", "/tmp"},
			},
			want: []string{"-oHostKeyAlias=i-0123456789abcdef0", "-lroot", "10.0.0.1", "--", "ls", "-la", "/tmp"},
		},
		{
			name: "EICE mode with proxy command",
			session: SSHSession{
				baseSSHSession: baseSSHSession{
					instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					destinationAddr: "i-0123456789abcdef0",
					Login:           "ec2-user",
					proxyCommand:    "ec2ssh --eice-tunnel",
				},
			},
			want: []string{"-oProxyCommand=ec2ssh --eice-tunnel", "-oHostKeyAlias=i-0123456789abcdef0", "-lec2-user", "i-0123456789abcdef0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.session.buildArgs()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSCPSessionBuildArgs(t *testing.T) {
	t.Parallel()

	instanceID := "i-0123456789abcdef0"

	tests := []struct {
		name    string
		session SCPSession
		want    []string
	}{
		{
			name: "upload local to remote",
			session: SCPSession{
				baseSSHSession: baseSSHSession{
					instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					destinationAddr: "10.0.0.1",
					Login:           "ec2-user",
				},
				LocalPath:  "/local/file.txt",
				RemotePath: "/remote/path/",
				IsUpload:   true,
			},
			want: []string{"-oHostKeyAlias=i-0123456789abcdef0", "/local/file.txt", "ec2-user@10.0.0.1:/remote/path/"},
		},
		{
			name: "download remote to local",
			session: SCPSession{
				baseSSHSession: baseSSHSession{
					instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					destinationAddr: "10.0.0.1",
					Login:           "ubuntu",
				},
				LocalPath:  "/tmp/download/",
				RemotePath: "/var/log/app.log",
				IsUpload:   false,
			},
			want: []string{"-oHostKeyAlias=i-0123456789abcdef0", "ubuntu@10.0.0.1:/var/log/app.log", "/tmp/download/"},
		},
		{
			name: "with port",
			session: SCPSession{
				baseSSHSession: baseSSHSession{
					instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					destinationAddr: "10.0.0.1",
					Login:           "root",
					Port:            "2222",
				},
				LocalPath:  "/local/file",
				RemotePath: "/remote/file",
				IsUpload:   true,
			},
			want: []string{"-oHostKeyAlias=i-0123456789abcdef0", "-P2222", "/local/file", "root@10.0.0.1:/remote/file"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.session.buildArgs()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSFTPSessionBuildArgs(t *testing.T) {
	t.Parallel()

	instanceID := "i-0123456789abcdef0"

	tests := []struct {
		name    string
		session SFTPSession
		want    []string
	}{
		{
			name: "basic SFTP connection",
			session: SFTPSession{
				baseSSHSession: baseSSHSession{
					instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					destinationAddr: "10.0.0.1",
					Login:           "ec2-user",
				},
			},
			want: []string{"-oHostKeyAlias=i-0123456789abcdef0", "ec2-user@10.0.0.1"},
		},
		{
			name: "with remote path",
			session: SFTPSession{
				baseSSHSession: baseSSHSession{
					instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					destinationAddr: "10.0.0.1",
					Login:           "ubuntu",
				},
				RemotePath: "/var/log",
			},
			want: []string{"-oHostKeyAlias=i-0123456789abcdef0", "ubuntu@10.0.0.1:/var/log"},
		},
		{
			name: "with port",
			session: SFTPSession{
				baseSSHSession: baseSSHSession{
					instance: types.Instance{
						InstanceId: aws.String(instanceID),
					},
					destinationAddr: "10.0.0.1",
					Login:           "admin",
					Port:            "2222",
				},
				RemotePath: "/home/admin",
			},
			want: []string{"-oHostKeyAlias=i-0123456789abcdef0", "-P2222", "admin@10.0.0.1:/home/admin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.session.buildArgs()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		dstTypeStr string
		wantErr    bool
	}{
		{name: "empty string is valid", dstTypeStr: "", wantErr: false},
		{name: "id is valid", dstTypeStr: "id", wantErr: false},
		{name: "private_ip is valid", dstTypeStr: "private_ip", wantErr: false},
		{name: "public_ip is valid", dstTypeStr: "public_ip", wantErr: false},
		{name: "ipv6 is valid", dstTypeStr: "ipv6", wantErr: false},
		{name: "private_dns is valid", dstTypeStr: "private_dns", wantErr: false},
		{name: "name_tag is valid", dstTypeStr: "name_tag", wantErr: false},
		{name: "invalid type returns error", dstTypeStr: "invalid", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			session := &baseSSHSession{DstTypeStr: tt.dstTypeStr}
			err := session.ParseTypes()
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "unknown")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	t.Parallel()

	t.Run("EICEID implies UseEICE", func(t *testing.T) {
		t.Parallel()
		session := &baseSSHSession{EICEID: "eice-12345678"}
		err := session.ApplyDefaults()
		assert.NoError(t, err)
		assert.True(t, session.UseEICE)
	})

	t.Run("empty EICEID does not set UseEICE", func(t *testing.T) {
		t.Parallel()
		session := &baseSSHSession{}
		err := session.ApplyDefaults()
		assert.NoError(t, err)
		assert.False(t, session.UseEICE)
	})

	t.Run("empty Login defaults to current user", func(t *testing.T) {
		t.Parallel()
		session := &baseSSHSession{}
		err := session.ApplyDefaults()
		assert.NoError(t, err)
		assert.NotEmpty(t, session.Login)
	})

	t.Run("non-empty Login is preserved", func(t *testing.T) {
		t.Parallel()
		session := &baseSSHSession{Login: "customuser"}
		err := session.ApplyDefaults()
		assert.NoError(t, err)
		assert.Equal(t, "customuser", session.Login)
	})
}
