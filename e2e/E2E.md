# E2E Test Matrix

This document describes the comprehensive end-to-end test coverage for ec2ssh.

## Test Dimensions

### 1. Intent (what user wants to do)

| Intent | Description | Test Pattern |
|--------|-------------|--------------|
| `ssh` | Execute remote command or open interactive shell | `ssh_*.txtar` |
| `scp` | Copy files to/from instance | `scp_*.txtar` |
| `sftp` | Interactive/batch file transfer | `sftp_*.txtar` |
| `ssm` | Open SSM shell (--ssm flag, no SSH) | `ssm*.txtar` |
| `list` | List instances (--list flag) | `list.txtar` |

### 2. Connect Method (how to reach the instance)

| Connect | Description | Flag | Network |
|---------|-------------|------|---------|
| `direct` | Direct SSH to public IP/hostname | (none) | Public IPv4 |
| `eice` | EICE tunnel with explicit endpoint ID | `--eice-id <id>` | Private |
| `eice_auto` | EICE tunnel with auto-detection | `--use-eice` | Private |
| `ssm` | SSM tunnel (SSH over SSM) | `--use-ssm` | Private/Public |
| `eice_ipv6_only` | EICE tunnel to IPv6-only instance | `--eice-id <id>` | IPv6-only VPC |

### 3. Key Management

| Feature | Description | Flag |
|---------|-------------|------|
| `autogen` | Generate ephemeral keypair, push via EC2IC | (default) |
| `identity` | Use provided private key, derive public, push via EC2IC | `-i <keyfile>` |
| `no-send` | Skip EC2IC key push (key pre-installed) | `--no-send-keys` |

### 4. Target Format

| Format | Example | Notes |
|--------|---------|-------|
| `simple` | `user@host` | Standard SSH format |
| `login_flag` | `-l user host` | SSH -l flag syntax |
| `url` | `ssh://user@host` | URL format (direct only) |

### 5. Destination Type (`--destination-type`)

| Type | Description | Example Input |
|------|-------------|---------------|
| `id` | EC2 Instance ID | `i-0123456789abcdef0` |
| `name_tag` | Instance Name tag | `my-instance` |
| `public_ip` | Public IPv4 address | `54.1.2.3` |
| `private_ip` | Private IPv4 address | `10.0.1.5` |
| `ipv6` | IPv6 address | `2600:1f18:...` |

### 6. Address Type (`--address-type`)

| Type | Description | Use Case |
|------|-------------|----------|
| `public` | Use public IPv4 | Direct internet access |
| `private` | Use private IPv4 | VPN/Direct Connect |
| `ipv6` | Use IPv6 address | IPv6-only or dual-stack |

## Test File Naming Convention

```
<intent>_<connect_method>[_<variant>].txtar
```

Examples:
- `ssh_direct.txtar` - SSH with direct connection
- `ssh_eice.txtar` - SSH via EICE tunnel (includes --eice-id and --use-eice)
- `ssh_eice_ipv6_only.txtar` - SSH via EICE to IPv6-only instance
- `scp_ssm.txtar` - SCP over SSM tunnel

## Current Test Coverage

### SSH Tests

| File | Connect | Key Mgmt | Target Format | Special |
|------|---------|----------|---------------|---------|
| `ssh_direct.txtar` | direct_public | autogen | simple, -l, url | --address-type, --destination-type |
| `ssh_identity.txtar` | direct_public | identity | simple | -i flag |
| `ssh_eice.txtar` | eice_private, eice_auto | autogen | simple, -l, url | --eice-id, --use-eice, --destination-type, --address-type |
| `ssh_eice_ipv6_only.txtar` | eice_ipv6, eice_auto | autogen | simple, -l | IPv6-only instance, --address-type ipv6 |
| `ssh_ssm.txtar` | ssm_tunnel | autogen | simple, -l, url | --use-ssm, --destination-type |

### SCP Tests

| File | Connect | Key Mgmt | Target Format | Special |
|------|---------|----------|---------------|---------|
| `scp_direct.txtar` | direct_public | autogen | simple, url | Upload/download |
| `scp_eice.txtar` | eice_private, eice_auto | autogen | simple | Upload/download via EICE |
| `scp_eice_ipv6_only.txtar` | eice_ipv6 | autogen | simple | IPv6-only instance |
| `scp_ssm.txtar` | ssm_tunnel | autogen | simple | Upload/download via SSM |

### SFTP Tests

| File | Connect | Key Mgmt | Target Format | Special |
|------|---------|----------|---------------|---------|
| `sftp_direct.txtar` | direct_public | autogen | simple, url | Batch mode |
| `sftp_eice.txtar` | eice_private, eice_auto | autogen | simple | Batch mode via EICE |
| `sftp_eice_ipv6_only.txtar` | eice_ipv6 | autogen | simple | IPv6-only instance |
| `sftp_ssm.txtar` | ssm_tunnel | autogen | simple | Batch mode via SSM |

### SSM Shell Tests

| File | Description |
|------|-------------|
| `ssm.txtar` | Basic --ssm shell to private instance |
| `ssm_targets.txtar` | --ssm with different target types (public, private IP) |

### Utility Tests

| File | Description |
|------|-------------|
| `list.txtar` | --list command with various --list-columns |
| `keys.txtar` | Key management: -i identity, --no-send-keys |
| `errors.txtar` | Error scenarios: invalid ID, invalid name, mutually exclusive flags |
| `passthrough.txtar` | Passthrough mode: ssh -V, --help |

## Coverage Matrix

### Intent × Transport Method

**Transports:**
- `direct_public` - Direct SSH to public IPv4 address
- `eice_private` - EICE tunnel to private IPv4 instance (`--eice-id`)
- `eice_auto` - EICE tunnel with auto-detection (`--use-eice`)
- `ssm_tunnel` - SSM Session Manager tunnel (`--use-ssm`)
- `eice_ipv6` - EICE tunnel to IPv6-only instance

| Intent | direct_public | eice_private | eice_auto | ssm_tunnel | eice_ipv6 |
|--------|---------------|--------------|-----------|------------|-----------|
| ssh | ✅ | ✅ | ✅ | ✅ | ✅ |
| scp | ✅ | ✅ | ✅ | ✅ | ✅ |
| sftp | ✅ | ✅ | ✅ | ✅ | ✅ |
| ssm (shell) | ✅ | N/A | N/A | N/A | N/A |
| list | ✅ | N/A | N/A | N/A | N/A |

**Note:** The `ssm` intent (`--ssm` flag) opens an SSM shell directly - it's not tunneled.
SSH/SCP/SFTP can use SSM as a tunnel via `--use-ssm`.

### Key Management Coverage

| Feature | Direct | EICE | SSM |
|---------|--------|------|-----|
| autogen (default) | ✅ | ✅ | ✅ |
| -i identity | ✅ | ✅ | ✅ |
| --no-send-keys | ✅ | ✅ | ✅ |

### Target Format Coverage

| Format | Direct | EICE | SSM |
|--------|--------|------|-----|
| user@host | ✅ | ✅ | ✅ |
| -l user host | ✅ | ✅ | ✅ |
| ssh://user@host | ✅ | ✅ | ✅ |

## Environment Variables

Tests use these environment variables (set in `e2e_test.go`):

### Standard VPC (Dual-Stack)

| Variable | Description |
|----------|-------------|
| `PUBLIC_ID` | Public instance ID |
| `PUBLIC_IP` | Public instance public IPv4 |
| `PUBLIC_NAME` | Public instance Name tag |
| `PRIVATE_ID` | Private instance ID |
| `PRIVATE_IP` | Private instance private IPv4 |
| `PRIVATE_NAME` | Private instance Name tag |
| `EICE_ID` | EC2 Instance Connect Endpoint ID |
| `USER` | SSH username (ec2-user) |

### IPv6-Only VPC

| Variable | Description |
|----------|-------------|
| `IPV6_ONLY_ID` | IPv6-only instance ID |
| `IPV6_ONLY_IPV6` | IPv6-only instance IPv6 address |
| `IPV6_ONLY_NAME` | IPv6-only instance Name tag |
| `EICE_IPV6_ID` | EICE ID in IPv6-only VPC |

## Adding New Tests

1. **Choose the right file**: Follow naming convention `<intent>_<connect>_<variant>.txtar`
2. **Use testscript format**: See [testscript documentation](https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript)
3. **Common SSH options**: Always use `-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null`
4. **Verify output**: Use `stdout 'expected'` or `stderr 'expected'`
5. **Error tests**: Prefix with `! exec` to expect failure

### Example Test Case

```txtar
# Description of what this test does

# Test case with explanation
exec ec2ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null $USER@$PUBLIC_ID -- echo test
stdout 'test'

# Test with -l flag syntax
exec ec2ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -l $USER $PUBLIC_ID -- hostname
stdout .
```

## Known Limitations

1. **SSM shell tests**: Require `expect` for PTY simulation, can be flaky
2. **IPv6 URL format**: OpenSSH's `valid_domain()` in `misc.c` rejects IPv6 addresses
   in `ssh://` URLs because colons (`:`) are not in the allowed character set.
   - `ssh ssh://[::1]:22` → **FAILS** (valid_domain rejects `::1`)
   - `ssh ssh://[2001:db8::1]:22` → **FAILS** (colons still invalid)
   - `ssh ::1` → **WORKS** (doesn't use URI parsing)
   - Use `user@ipv6` format instead of `ssh://user@[ipv6]`
3. **EICE auto-detection**: Finds EICE in instance's VPC, prefers same subnet, falls back to first found
