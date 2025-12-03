package cli

import (
	"errors"
	"fmt"
)

var (
	// ErrSCP is the parent error for all SCP parsing errors.
	ErrSCP = errors.New("scp error")

	// ErrSCPTooFewOperands indicates less than 2 operands were provided.
	ErrSCPTooFewOperands = fmt.Errorf("%w: requires exactly 2 operands (source and destination)", ErrSCP)

	// ErrSCPTooManyOperands indicates more than 2 operands were provided.
	ErrSCPTooManyOperands = fmt.Errorf("%w: requires exactly 2 operands (source and destination)", ErrSCP)

	// ErrSCPNoRemote indicates neither operand is remote (no colon found).
	ErrSCPNoRemote = fmt.Errorf("%w: no remote operand found (use host:path syntax)", ErrSCP)

	// ErrSCPMultipleRemotes indicates both operands are remote.
	ErrSCPMultipleRemotes = fmt.Errorf("%w: multiple remote operands not supported", ErrSCP)

	// ErrSCPEmptyPath indicates the path after ':' is empty.
	ErrSCPEmptyPath = fmt.Errorf("%w: remote path cannot be empty after ':'", ErrSCP)

	// ErrSCPEmptyHost indicates the host before ':' is empty.
	ErrSCPEmptyHost = fmt.Errorf("%w: remote host cannot be empty", ErrSCP)
)
