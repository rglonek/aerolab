package cmd

import (
	"reflect"
	"strings"
)

// redactedPlaceholder replaces the value of any field tagged for redaction.
const redactedPlaceholder = "<redacted>"

// telemetryRedactTag marks a struct field whose value must never leave the
// machine. Tag secret-bearing command fields with `telemetry:"redact"`.
const telemetryRedactTag = "telemetry"

// redactedFlags describes the command-line flags belonging to redacted fields,
// so the raw os.Args copy can be scrubbed to match.
type redactedFlags struct {
	long  map[string]bool // flag names without the leading "--"
	short map[string]bool // single-character names without the leading "-"
}

func newRedactedFlags() *redactedFlags {
	return &redactedFlags{long: map[string]bool{}, short: map[string]bool{}}
}

func (r *redactedFlags) empty() bool {
	return len(r.long) == 0 && len(r.short) == 0
}

// redactParams deep-copies a command struct into a plain map/slice tree with
// every field tagged `telemetry:"redact"` replaced by a placeholder, and
// returns the flag names of those fields.
//
// A copy is made rather than mutating in place because the caller is the live
// command struct, still in use by the running command. Fields the JSON encoder
// would skip (unexported, `json:"-"`) are dropped here too, so the result
// matches what would otherwise have been shipped.
func redactParams(params any) (any, *redactedFlags) {
	flags := newRedactedFlags()
	if params == nil {
		return nil, flags
	}
	return redactValue(reflect.ValueOf(params), false, "", flags), flags
}

// redactValue walks v, returning a redaction-safe copy.
//
// redact is set once an enclosing field was tagged, so nested structs behind a
// tagged field are scrubbed wholesale. namespace carries the go-flags group
// prefix so nested flag names are recorded in the same form the user typed.
func redactValue(v reflect.Value, redact bool, namespace string, flags *redactedFlags) any {
	if !v.IsValid() {
		return nil
	}

	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return redactValue(v.Elem(), redact, namespace, flags)

	case reflect.Struct:
		return redactStruct(v, redact, namespace, flags)

	case reflect.Slice, reflect.Array:
		if v.Kind() == reflect.Slice && v.IsNil() {
			return nil
		}
		// []byte marshals as a base64 string; keep that shape.
		if v.Type().Elem().Kind() == reflect.Uint8 {
			if redact {
				return redactedPlaceholder
			}
			return v.Interface()
		}
		out := make([]any, 0, v.Len())
		for i := 0; i < v.Len(); i++ {
			out = append(out, redactValue(v.Index(i), redact, namespace, flags))
		}
		return out

	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		out := make(map[string]any, v.Len())
		for _, key := range v.MapKeys() {
			out[stringifyMapKey(key)] = redactValue(v.MapIndex(key), redact, namespace, flags)
		}
		return out

	default:
		if redact && !v.IsZero() {
			return redactedPlaceholder
		}
		if !v.CanInterface() {
			return nil
		}
		return v.Interface()
	}
}

func redactStruct(v reflect.Value, redact bool, namespace string, flags *redactedFlags) any {
	t := v.Type()
	out := make(map[string]any, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			// Unexported: json.Marshal would have skipped it anyway.
			continue
		}
		if jsonTag := field.Tag.Get("json"); jsonTag == "-" {
			continue
		}

		fieldRedact := redact || field.Tag.Get(telemetryRedactTag) == "redact"

		fieldNamespace := namespace
		if ns := field.Tag.Get("namespace"); ns != "" {
			fieldNamespace = ns + "."
		}

		if fieldRedact {
			recordRedactedFlag(field, fieldNamespace, flags)
		}

		// Embedded structs are flattened by go-flags and by json, so flatten
		// them here too rather than nesting them under the type name.
		if field.Anonymous && dereferenceKind(field.Type) == reflect.Struct {
			nested := redactValue(v.Field(i), fieldRedact, fieldNamespace, flags)
			if nestedMap, ok := nested.(map[string]any); ok {
				for k, val := range nestedMap {
					out[k] = val
				}
				continue
			}
		}

		out[jsonFieldName(field)] = redactValue(v.Field(i), fieldRedact, fieldNamespace, flags)
	}
	return out
}

func dereferenceKind(t reflect.Type) reflect.Kind {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Kind()
}

func recordRedactedFlag(field reflect.StructField, namespace string, flags *redactedFlags) {
	if long := field.Tag.Get("long"); long != "" {
		flags.long[namespace+long] = true
	}
	if short := field.Tag.Get("short"); short != "" {
		flags.short[short] = true
	}
}

// jsonFieldName returns the key json.Marshal would have used for the field.
func jsonFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" {
		return field.Name
	}
	return name
}

func stringifyMapKey(key reflect.Value) string {
	if key.Kind() == reflect.String {
		return key.String()
	}
	if s, ok := key.Interface().(interface{ String() string }); ok {
		return s.String()
	}
	return reflect.ValueOf(key.Interface()).String()
}

// redactCmdLine returns a copy of args with the values of redacted flags
// replaced, covering the `--flag=v`, `--flag v`, `-x v` and `-xv` forms.
//
// The struct-tag walk only protects Params; without this, the same secret
// would still ship verbatim in the recorded command line.
func redactCmdLine(args []string, flags *redactedFlags) []string {
	if flags == nil || flags.empty() || len(args) == 0 {
		return args
	}
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Everything after a bare "--" is positional.
		if arg == "--" {
			out = append(out, args[i:]...)
			break
		}

		if name, value, hasValue := strings.Cut(arg, "="); hasValue && strings.HasPrefix(arg, "--") {
			if flags.long[strings.TrimPrefix(name, "--")] {
				out = append(out, name+"="+redactedPlaceholder)
				continue
			}
			_ = value
			out = append(out, arg)
			continue
		}

		if strings.HasPrefix(arg, "--") {
			if flags.long[strings.TrimPrefix(arg, "--")] {
				out = append(out, arg)
				if i+1 < len(args) {
					out = append(out, redactedPlaceholder)
					i++
				}
				continue
			}
			out = append(out, arg)
			continue
		}

		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			short := string(arg[1])
			if flags.short[short] {
				if len(arg) > 2 {
					// Attached value, e.g. -pSECRET.
					out = append(out, "-"+short+redactedPlaceholder)
					continue
				}
				out = append(out, arg)
				if i+1 < len(args) {
					out = append(out, redactedPlaceholder)
					i++
				}
				continue
			}
		}

		out = append(out, arg)
	}
	return out
}

// sensitiveDefaultKeySubstrings match config keys whose saved value is
// secret-shaped. Defaults are reported as flat "Section.Field" keys with no
// access to the originating struct tag, so they are matched by name.
var sensitiveDefaultKeySubstrings = []string{
	"password",
	"passwd",
	"pass",
	"secret",
	"token",
	"credential",
	"apikey",
	"api-key",
	"privatekey",
	"private-key",
	"keyid",
	"key-id",
	"username",
	"user",
}

// isSensitiveDefaultKey reports whether a config default key should be dropped
// from telemetry.
func isSensitiveDefaultKey(key string) bool {
	// Only the final path element is examined, so that a section named e.g.
	// "Backend" does not drag in every field beneath it.
	leaf := key
	if idx := strings.LastIndex(key, "."); idx >= 0 {
		leaf = key[idx+1:]
	}
	leaf = strings.ToLower(leaf)
	for _, s := range sensitiveDefaultKeySubstrings {
		if strings.Contains(leaf, s) {
			return true
		}
	}
	return false
}

// sensitiveEnvVarSubstrings match AEROLAB_* environment variable names whose
// value should be masked rather than shipped.
var sensitiveEnvVarSubstrings = []string{
	"PASSWORD",
	"PASSWD",
	"PASS",
	"SECRET",
	"TOKEN",
	"CREDENTIAL",
	"APIKEY",
	"API_KEY",
	"PRIVATE_KEY",
	"KEY_ID",
	"AUTH",
}

// redactEnvVars masks the values of secret-shaped environment variables in
// place. Users routinely pass secrets through ENV::VARNAME indirection, so
// these variables are as sensitive as the flags that reference them.
func redactEnvVars(envVars map[string]string) map[string]string {
	for key := range envVars {
		upper := strings.ToUpper(key)
		for _, s := range sensitiveEnvVarSubstrings {
			if strings.Contains(upper, s) {
				envVars[key] = redactedPlaceholder
				break
			}
		}
	}
	return envVars
}
