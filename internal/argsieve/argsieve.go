// Package argsieve provides argument parsing with two modes:
//   - Sift: extracts known flags, passes unknown flags through (for CLI wrappers)
//   - Parse: strict parsing that errors on unknown flags
package argsieve

import (
	"errors"
	"fmt"
	"iter"
	"reflect"
	"slices"
	"strings"
)

// ErrParse indicates a parsing error (e.g., missing value or unknown option).
var ErrParse = errors.New("argument parsing error")

// fieldInfo holds a reference to a struct field and whether it needs an argument.
type fieldInfo struct {
	field    reflect.Value
	needsArg bool
}

// sieve separates known flags from unknown flags and positional arguments.
type sieve struct {
	fields      map[string]fieldInfo // flag name → field info
	passthrough map[string]struct{}
	remaining   []string
	positional  []string
	strict      bool
}

// Sift extracts known flags into target, returning unknown flags and positional args.
// passthroughWithArg lists unknown flags that take values (e.g., []string{"-o", "-L"}).
//
// Panics if target is not a pointer to struct or if any tagged field
// has a type other than string or bool.
func Sift(target any, args []string, passthroughWithArg []string) (remaining, positional []string, err error) {
	s := &sieve{
		fields:      make(map[string]fieldInfo),
		passthrough: make(map[string]struct{}),
	}

	s.extractFields(target)

	for _, p := range passthroughWithArg {
		s.passthrough[p] = struct{}{}
	}

	return s.parse(args)
}

// Parse parses args into target, returning positional args.
// Returns error if unknown flags are encountered.
//
// Panics if target is not a pointer to struct or if any tagged field
// has a type other than string or bool.
func Parse(target any, args []string) (positional []string, err error) {
	s := &sieve{
		fields:      make(map[string]fieldInfo),
		passthrough: make(map[string]struct{}),
		strict:      true,
	}

	s.extractFields(target)

	_, positional, err = s.parse(args)

	return positional, err
}

// Helper methods for cleaner append patterns.
func (s *sieve) addRemaining(args ...string)  { s.remaining = append(s.remaining, args...) }
func (s *sieve) addPositional(args ...string) { s.positional = append(s.positional, args...) }

// extractFields reads struct tags and stores field references.
// Panics if target is not a pointer to a struct.
func (s *sieve) extractFields(target any) {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("argsieve: target must be a pointer to struct, got %T", target))
	}

	s.extractFieldsFromValue(v.Elem())
}

// extractFieldsFromValue recursively extracts fields from a struct value,
// including fields from embedded structs.
func (s *sieve) extractFieldsFromValue(v reflect.Value) {
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		fieldType := t.Field(i)
		fieldValue := v.Field(i)

		// Recursively process embedded structs
		if fieldType.Anonymous && fieldType.Type.Kind() == reflect.Struct {
			s.extractFieldsFromValue(fieldValue)
			continue
		}

		short := fieldType.Tag.Get("short")
		long := fieldType.Tag.Get("long")

		// Skip fields without tags
		if short == "" && long == "" {
			continue
		}

		// Validate field type - only string and bool are supported
		kind := fieldType.Type.Kind()
		if kind != reflect.String && kind != reflect.Bool {
			panic(fmt.Sprintf("argsieve: field %s has unsupported type %s (only string and bool are supported)",
				fieldType.Name, fieldType.Type))
		}

		info := fieldInfo{field: fieldValue, needsArg: kind != reflect.Bool}

		if short != "" {
			s.fields[short] = info
		}

		if long != "" {
			s.fields[long] = info
		}
	}
}

// setField assigns a value to a field based on its type.
func (s *sieve) setField(info fieldInfo, value string) {
	if info.needsArg {
		info.field.SetString(value)
	} else {
		info.field.SetBool(true)
	}
}

// handleLong processes --name or --name=value arguments.
func (s *sieve) handleLong(arg string, next func() (string, bool)) error {
	name, eqValue, hasEquals := strings.Cut(arg[2:], "=")

	info, known := s.fields[name]

	// Unknown flag - reject in strict mode or check passthrough list
	if !known {
		if s.strict {
			return fmt.Errorf("%w: unknown option --%s", ErrParse, name)
		}

		_, isPassthrough := s.passthrough["--"+name]

		if isPassthrough && !hasEquals {
			if value, ok := next(); ok {
				s.addRemaining(arg, value)

				return nil
			}
		}

		s.addRemaining(arg)

		return nil
	}

	// Known bool flag
	if !info.needsArg {
		s.setField(info, "")

		return nil
	}

	// Known string flag with equals
	if hasEquals {
		s.setField(info, eqValue)

		return nil
	}

	// Known string flag - needs argument from next arg
	value, ok := next()
	if !ok {
		return fmt.Errorf("%w: missing value for --%s", ErrParse, name)
	}

	s.setField(info, value)

	return nil
}

// handleShort processes -x, -xvalue, or -xyz combined arguments.
func (s *sieve) handleShort(arg string, next func() (string, bool)) error {
	flags := arg[1:]

	for j := 0; j < len(flags); j++ {
		flag := string(flags[j])
		tail := flags[j+1:]

		info, known := s.fields[flag]

		// Handle unknown flag first (guard clause)
		if !known {
			if err := s.handleUnknownShort(flag, tail, next); err != nil {
				return err
			}

			if len(tail) > 0 {
				return nil // tail consumed by passthrough
			}

			continue
		}

		// Known bool flag
		if !info.needsArg {
			s.setField(info, "")

			continue
		}

		// Known string flag - value attached
		if len(tail) > 0 {
			s.setField(info, tail)

			return nil
		}

		// Known string flag - value in next arg
		value, ok := next()
		if !ok {
			return fmt.Errorf("%w: missing value for -%s", ErrParse, flag)
		}

		s.setField(info, value)

		return nil
	}

	return nil
}

// handleUnknownShort handles unknown short flags, checking passthrough list.
func (s *sieve) handleUnknownShort(flag, tail string, next func() (string, bool)) error {
	if s.strict {
		return fmt.Errorf("%w: unknown option -%s", ErrParse, flag)
	}

	prefixedFlag := "-" + flag
	_, isPassthrough := s.passthrough[prefixedFlag]

	if isPassthrough {
		if len(tail) > 0 {
			s.addRemaining("-" + flag + tail)

			return nil
		}

		if value, ok := next(); ok {
			s.addRemaining(prefixedFlag, value)

			return nil
		}
	}

	s.addRemaining(prefixedFlag)

	return nil
}

// parse separates args into known flags (bound to target), unknown flags, and positionals.
// Arguments after "--" are treated as positional (the "--" itself is not included).
func (s *sieve) parse(args []string) (remaining, positional []string, err error) {
	next, stop := iter.Pull(slices.Values(args))
	defer stop()

	for arg, ok := next(); ok; arg, ok = next() {
		switch {
		case arg == "--":
			// Drain remaining args as positional (don't pass "--" through)
			for arg, ok := next(); ok; arg, ok = next() {
				s.addPositional(arg)
			}

		case strings.HasPrefix(arg, "--"):
			if err := s.handleLong(arg, next); err != nil {
				return nil, nil, err
			}

		case strings.HasPrefix(arg, "-") && len(arg) > 1:
			if err := s.handleShort(arg, next); err != nil {
				return nil, nil, err
			}

		default:
			s.addPositional(arg)
		}
	}

	return s.remaining, s.positional, nil
}
