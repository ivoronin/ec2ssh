package main

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var (
	ErrArgParse = errors.New("error parsing arguments")
	ErrHelp     = errors.New("help requested")
)

type ParsedArgs struct {
	Options         map[string]string
	Destination     string
	CommandWithArgs []string
	SSHArgs         []string
}

func parseLongOption(args []string, idx int, parsedArgs *ParsedArgs) (int, error) {
	consumedLongOptionsWithArguments := []string{
		"--list-columns",
		"--region",
		"--profile",
		"--destination-type",
		"--address-type",
		"--eice-id",
	}
	consumedLongOptionsWithoutArguments := []string{
		"--list",
		"--no-send-keys",
		"--use-eice",
		"--debug",
	}

	arg := args[idx]
	if arg == "--help" {
		return idx, ErrHelp
	}

	if slices.Contains(consumedLongOptionsWithoutArguments, arg) {
		if _, ok := parsedArgs.Options[arg]; !ok {
			parsedArgs.Options[arg] = "true"
		}

		return idx, nil
	}

	option, value, includesValue := strings.Cut(arg, "=")
	if slices.Contains(consumedLongOptionsWithArguments, option) {
		if !includesValue {
			/* value is in the next argument */
			if idx+1 >= len(args) {
				return idx, fmt.Errorf("%w: missing value for %s", ErrArgParse, arg)
			}

			value = args[idx+1]
			idx++
		}

		if _, ok := parsedArgs.Options[arg]; !ok {
			parsedArgs.Options[option] = value
		}

		return idx, nil
	}

	/* SSH doesn't support long options, so we error out here */
	return idx, fmt.Errorf("%w: unknown option %s", ErrArgParse, arg)
}

func parseShortOption(args []string, idx int, parsedArgs *ParsedArgs) (int, error) {
	consumedOptionsWithArguments := "lpi"
	unconsumedOptionsWithArguments := "BbcDEeFIiJLlmOoPpRSWw"
	optionsWithArguments := consumedOptionsWithArguments + unconsumedOptionsWithArguments
	flags := args[idx][1:]

	for flagIdx := 0; flagIdx < len(flags); flagIdx++ {
		flag := string(flags[flagIdx])

		if flag == "h" {
			return idx, ErrHelp
		}

		/* we don't consume any options without arguments */
		if !strings.Contains(optionsWithArguments, flag) {
			parsedArgs.SSHArgs = append(parsedArgs.SSHArgs, "-"+flag)

			continue
		}

		var value string

		/* value is attached to the current flag */
		if flagIdx+1 < len(flags) {
			value = flags[flagIdx+1:]
			flagIdx = len(flags) // Stop iterating over current argument
		} else {
			/* value is in the next argument */
			if idx+1 >= len(args) {
				return idx, fmt.Errorf("%w: missing value for %s", ErrArgParse, args[idx])
			}

			value = args[idx+1]
			idx++
		}

		if strings.Contains(consumedOptionsWithArguments, flag) {
			if _, ok := parsedArgs.Options["-"+flag]; !ok {
				parsedArgs.Options["-"+flag] = value
			}
		} else {
			parsedArgs.SSHArgs = append(parsedArgs.SSHArgs, "-"+flag+value)
		}
	}

	return idx, nil
}

func ParseArgs(args []string) (ParsedArgs, error) {
	var err error

	parsedArgs := ParsedArgs{
		Options:         make(map[string]string),
		Destination:     "",
		CommandWithArgs: []string{},
		SSHArgs:         []string{},
	}

loop:
	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]

		switch {
		case arg == "--":
			parsedArgs.CommandWithArgs = args[idx+1:]

			break loop
		case strings.HasPrefix(arg, "--"): /* long option */
			idx, err = parseLongOption(args, idx, &parsedArgs)
			if err != nil {
				return ParsedArgs{}, err
			}
		case strings.HasPrefix(arg, "-") && len(arg) > 1: /* short option */
			idx, err = parseShortOption(args, idx, &parsedArgs)
			if err != nil {
				return ParsedArgs{}, err
			}
		default: /* non-option argument */
			if parsedArgs.Destination == "" {
				parsedArgs.Destination = arg
			} else {
				parsedArgs.CommandWithArgs = args[idx:]

				break loop
			}
		}
	}

	return parsedArgs, nil
}
