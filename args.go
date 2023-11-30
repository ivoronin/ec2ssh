package main

import (
	"os"
	"os/user"
	"strings"
)

type DstType int

const (
	DstTypeUnknown DstType = iota
	DstTypeID
	DstTypePrivateIP
	DstTypePublicIP
	DstTypePrivateDNSName
	DstTypeNameTag
)

var DstTypeNames = map[string]DstType{
	"id":          DstTypeID,
	"private_ip":  DstTypePrivateIP,
	"public_ip":   DstTypePublicIP,
	"private_dns": DstTypePrivateDNSName,
	"name_tag":    DstTypeNameTag,
}

type Opts struct {
	loginUser        string
	sshPublicKeyPath string
	usePublicIP      bool
	dstType          DstType
	region           string
	profile          string
}

type SSHArgs struct {
	args   []string
	dstIdx int
}

func (a SSHArgs) Destination() string {
	return a.args[a.dstIdx]
}

func (a SSHArgs) SetDestination(dst string) {
	a.args[a.dstIdx] = dst
}

func (a SSHArgs) Args() []string {
	return a.args
}

type ArgShifter struct {
	slice  []string
	index  int
	length int
}

func NewArgShifter(slice *[]string) ArgShifter {
	return ArgShifter{
		slice:  *slice,
		index:  0,
		length: len(*slice),
	}
}

func (s *ArgShifter) Try() *string {
	if s.index < s.length {
		elem := &s.slice[s.index]
		s.index++
		return elem
	}
	return nil
}

func (s *ArgShifter) Must() string {
	elem := s.Try()
	if elem == nil {
		Usage()
	}
	return *elem
}

func ParseArgs() (Opts, SSHArgs) {
	args := os.Args[1:]
	if len(args) < 1 {
		Usage()
	}

	usr, err := user.Current()
	if err != nil {
		HandleError(err)
	}

	/* default values */
	opts := Opts{
		loginUser:        "ec2-user",
		sshPublicKeyPath: usr.HomeDir + "/.ssh/id_rsa.pub",
		usePublicIP:      false,
		dstType:          DstTypeUnknown,
	}

	sshArgs := SSHArgs{
		args:   make([]string, 0, len(args)),
		dstIdx: -1,
	}

	shifter := NewArgShifter(&args)
	for argp := shifter.Try(); argp != nil; argp = shifter.Try() {
		arg := *argp
		/* ssh doesn't use long keys, so we do */
		if strings.HasPrefix(arg, "--") && len(arg) > 2 {
			switch arg {
			case "--public-key":
				opts.sshPublicKeyPath = shifter.Must()
			case "--region":
				opts.region = shifter.Must()
			case "--profile":
				opts.profile = shifter.Must()
			case "--use-public-ip":
				opts.usePublicIP = true
			case "--destination-type":
				dstType := shifter.Must()
				var ok bool
				opts.dstType, ok = DstTypeNames[dstType]
				if !ok {
					Usage()
				}
			default:
				Usage()
			}
			continue
		}

		sshArgs.args = append(sshArgs.args, arg)
		if arg == "-l" {
			opts.loginUser = shifter.Must()
			sshArgs.args = append(sshArgs.args, opts.loginUser)
		} else if !strings.HasPrefix(arg, "-") && sshArgs.dstIdx == -1 {
			sshArgs.dstIdx = len(sshArgs.args) - 1
		}
	}

	if sshArgs.dstIdx == -1 {
		Usage()
	}

	return opts, sshArgs
}
