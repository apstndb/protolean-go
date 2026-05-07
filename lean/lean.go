// Package lean implements the LEAN (LLM-Efficient Adaptive Notation) format
// encoder and decoder. LEAN is a token-optimized serialization format that
// uses tab delimiters, single-char keywords (T/F/_), and tabular arrays.
package lean

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Value represents any JSON-compatible value.
type Value = any

// Marshal encodes a Go value as a LEAN document.
func Marshal(v Value) ([]byte, error) {
	s, err := Encode(v)
	if err != nil {
		return nil, err
	}
	return []byte(s), nil
}

// MarshalString encodes v as a LEAN document string.
func MarshalString(v Value) (string, error) {
	return Encode(v)
}

// Encode encodes a Go value as a LEAN string.
func Encode(v Value) (string, error) {
	lines, err := encodeValue(v, "", 0)
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

var keyRegex = regexp.MustCompile(`^[\w][\w-]*$`)
var numberRegex = regexp.MustCompile(`^-?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$`)

func validateKey(key string) error {
	if !keyRegex.MatchString(key) {
		return fmt.Errorf("lean: unsupported key %q: keys must match [\\w][\\w-]*", key)
	}
	return nil
}

func isScalar(v Value) bool {
	if v == nil {
		return true
	}
	switch v.(type) {
	case string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64, bool:
		return true
	}
	return false
}

func isTabularArray(arr []Value) bool {
	if len(arr) == 0 {
		return false
	}
	var firstKeys []string
	for i, item := range arr {
		obj, ok := item.(map[string]Value)
		if !ok || obj == nil {
			return false
		}
		if i == 0 {
			firstKeys = make([]string, 0, len(obj))
			for k := range obj {
				firstKeys = append(firstKeys, k)
			}
		}
		if len(obj) != len(firstKeys) {
			return false
		}
		for _, k := range firstKeys {
			if _, ok := obj[k]; !ok {
				return false
			}
			if !isScalar(obj[k]) {
				return false
			}
		}
	}
	return true
}

func needsQuoting(value string) bool {
	if value == "" {
		return true
	}
	if value == "T" || value == "F" || value == "_" {
		return true
	}
	if strings.TrimSpace(value) != value {
		return true
	}
	if numberRegex.MatchString(value) {
		return true
	}
	if strings.Contains(value, "\t") || strings.Contains(value, "\n") || strings.Contains(value, "\\") || strings.Contains(value, `"`) {
		return true
	}
	return false
}

func escapeScalar(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

func encodeScalar(value string, forceQuote bool) string {
	if forceQuote || needsQuoting(value) {
		return fmt.Sprintf(`"%s"`, escapeScalar(value))
	}
	return value
}

func escapeCell(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	value = strings.ReplaceAll(value, `"`, `""`)
	return value
}

func cellEncode(v Value) string {
	switch val := v.(type) {
	case nil:
		return "_"
	case bool:
		if val {
			return "T"
		}
		return "F"
	case int:
		return strconv.FormatInt(int64(val), 10)
	case int8:
		return strconv.FormatInt(int64(val), 10)
	case int16:
		return strconv.FormatInt(int64(val), 10)
	case int32:
		return strconv.FormatInt(int64(val), 10)
	case int64:
		return strconv.FormatInt(val, 10)
	case uint:
		return strconv.FormatUint(uint64(val), 10)
	case uint8:
		return strconv.FormatUint(uint64(val), 10)
	case uint16:
		return strconv.FormatUint(uint64(val), 10)
	case uint32:
		return strconv.FormatUint(uint64(val), 10)
	case uint64:
		return strconv.FormatUint(val, 10)
	case float32:
		return strconv.FormatFloat(float64(val), 'f', -1, 32)
	case float64:
		return strconv.FormatFloat(val, 'f', -1, 64)
	case string:
		if needsQuoting(val) {
			return fmt.Sprintf(`"%s"`, escapeCell(val))
		}
		return val
	default:
		panic(fmt.Sprintf("lean: cellEncode called with non-scalar %T", v))
	}
}

func encodePrimitive(v Value) (string, error) {
	switch val := v.(type) {
	case nil:
		return "_", nil
	case bool:
		if val {
			return "T", nil
		}
		return "F", nil
	case int:
		return strconv.FormatInt(int64(val), 10), nil
	case int8:
		return strconv.FormatInt(int64(val), 10), nil
	case int16:
		return strconv.FormatInt(int64(val), 10), nil
	case int32:
		return strconv.FormatInt(int64(val), 10), nil
	case int64:
		return strconv.FormatInt(val, 10), nil
	case uint:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint8:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(val), 10), nil
	case uint64:
		return strconv.FormatUint(val, 10), nil
	case float32:
		if math.IsNaN(float64(val)) || math.IsInf(float64(val), 0) {
			return "", fmt.Errorf("lean: unsupported number: %v", val)
		}
		return strconv.FormatFloat(float64(val), 'f', -1, 32), nil
	case float64:
		if math.IsNaN(val) || math.IsInf(val, 0) {
			return "", fmt.Errorf("lean: unsupported number: %v", val)
		}
		return strconv.FormatFloat(val, 'f', -1, 64), nil
	case string:
		return encodeScalar(val, false), nil
	default:
		return "", fmt.Errorf("lean: not a primitive: %T", v)
	}
}

func encodeValue(v Value, path string, indent int) ([]string, error) {
	pad := strings.Repeat("  ", indent)

	if isScalar(v) {
		s, err := encodePrimitive(v)
		if err != nil {
			return nil, err
		}
		if path == "" {
			// Root scalar: strings must always be quoted
			if str, ok := v.(string); ok {
				s = encodeScalar(str, true)
			}
			return []string{s}, nil
		}
		return []string{fmt.Sprintf("%s%s:%s", pad, path, s)}, nil
	}

	switch val := v.(type) {
	case []Value:
		return encodeArray(val, path, indent)
	case map[string]Value:
		if len(val) == 0 {
			if path == "" {
				return []string{"{}"}, nil
			}
			return []string{fmt.Sprintf("%s%s:{}", pad, path)}, nil
		}
		return encodeObject(val, path, indent)
	default:
		return nil, fmt.Errorf("lean: unsupported type %T", v)
	}
}

func encodeObject(obj map[string]Value, path string, indent int) ([]string, error) {
	if indent == 0 && path != "" {
		// At root level with a path prefix (nested object encoding)
		// Try dot-flatten vs indented block, pick shorter
		dotLines, err := encodeObjectDotFlatten(obj, path)
		if err != nil {
			return nil, err
		}
		blockLines, err := encodeObjectBlock(obj, path, 0)
		if err != nil {
			return nil, err
		}
		dotCost := len(strings.Join(dotLines, "\n"))
		blockCost := len(strings.Join(blockLines, "\n"))
		if dotCost <= blockCost {
			return dotLines, nil
		}
		return blockLines, nil
	}

	return encodeObjectBlock(obj, path, indent)
}

func encodeObjectDotFlatten(obj map[string]Value, prefix string) ([]string, error) {
	var lines []string
	for key, val := range obj {
		if err := validateKey(key); err != nil {
			return nil, err
		}
		subLines, err := encodeValue(val, fmt.Sprintf("%s.%s", prefix, key), 0)
		if err != nil {
			return nil, err
		}
		lines = append(lines, subLines...)
	}
	return lines, nil
}

func encodeObjectBlock(obj map[string]Value, path string, indent int) ([]string, error) {
	pad := strings.Repeat("  ", indent)
	var lines []string
	if path != "" {
		lines = append(lines, fmt.Sprintf("%s%s:", pad, path))
		indent++
		pad = strings.Repeat("  ", indent)
	}
	for key, val := range obj {
		if err := validateKey(key); err != nil {
			return nil, err
		}
		subLines, err := encodeValue(val, key, indent)
		if err != nil {
			return nil, err
		}
		lines = append(lines, subLines...)
	}
	return lines, nil
}

func encodeArray(arr []Value, path string, indent int) ([]string, error) {
	pad := strings.Repeat("  ", indent)
	prefix := path

	if len(arr) == 0 {
		return []string{fmt.Sprintf("%s%s[0]:", pad, prefix)}, nil
	}

	// Flat scalar array
	allScalar := true
	for _, v := range arr {
		if !isScalar(v) {
			allScalar = false
			break
		}
	}
	if allScalar {
		cells := make([]string, len(arr))
		for i, v := range arr {
			cells[i] = cellEncode(v)
		}
		return []string{fmt.Sprintf("%s%s[%d]:%s", pad, prefix, len(arr), strings.Join(cells, "\t"))}, nil
	}

	// Tabular array
	if isTabularArray(arr) {
		fields := make([]string, 0, len(arr[0].(map[string]Value)))
		for k := range arr[0].(map[string]Value) {
			if err := validateKey(k); err != nil {
				return nil, err
			}
			fields = append(fields, k)
		}
		lines := []string{fmt.Sprintf("%s%s[%d]:%s", pad, prefix, len(arr), strings.Join(fields, "\t"))}
		for _, row := range arr {
			obj := row.(map[string]Value)
			cells := make([]string, len(fields))
			for i, f := range fields {
				cells[i] = cellEncode(obj[f])
			}
			lines = append(lines, fmt.Sprintf("%s  %s", pad, strings.Join(cells, "\t")))
		}
		return lines, nil
	}

	// Semi-tabular: all items are objects with all-scalar values but different keys
	allObjects := true
	for _, item := range arr {
		obj, ok := item.(map[string]Value)
		if !ok || obj == nil {
			allObjects = false
			break
		}
		allScalarValues := true
		for _, v := range obj {
			if !isScalar(v) {
				allScalarValues = false
				break
			}
		}
		if !allScalarValues {
			allObjects = false
			break
		}
	}

	if allObjects && len(arr) >= 2 {
		objects := make([]map[string]Value, len(arr))
		for i, item := range arr {
			objects[i] = item.(map[string]Value)
		}

		// Find shared keys
		var sharedKeys []string
		for k := range objects[0] {
			allHave := true
			for _, obj := range objects[1:] {
				if _, ok := obj[k]; !ok {
					allHave = false
					break
				}
			}
			if allHave {
				sharedKeys = append(sharedKeys, k)
			}
		}

		if len(sharedKeys) > 0 {
			for _, k := range sharedKeys {
				if err := validateKey(k); err != nil {
					return nil, err
				}
			}
			sharedSet := make(map[string]struct{})
			for _, k := range sharedKeys {
				sharedSet[k] = struct{}{}
			}

			// Build semi-tabular encoding
			semiLines := []string{fmt.Sprintf("%s%s[%d]:%s\t~", pad, prefix, len(arr), strings.Join(sharedKeys, "\t"))}
			for _, obj := range objects {
				factored := make([]string, len(sharedKeys))
				for i, k := range sharedKeys {
					factored[i] = cellEncode(obj[k])
				}
				var remaining []string
				for k, v := range obj {
					if _, ok := sharedSet[k]; !ok {
						if err := validateKey(k); err != nil {
							return nil, err
						}
						remaining = append(remaining, fmt.Sprintf("%s:%s", k, cellEncode(v)))
					}
				}
				cells := append(factored, remaining...)
				semiLines = append(semiLines, fmt.Sprintf("%s  %s", pad, strings.Join(cells, "\t")))
			}

			// Build dashed-list encoding for comparison
			dashedLines := []string{fmt.Sprintf("%s%s[%d]:", pad, prefix, len(arr))}
			for _, item := range arr {
				subLines, err := encodeListItem(item, indent+1)
				if err != nil {
					return nil, err
				}
				dashedLines = append(dashedLines, subLines...)
			}

			semiCost := len(strings.Join(semiLines, "\n"))
			dashedCost := len(strings.Join(dashedLines, "\n"))
			if semiCost < dashedCost {
				return semiLines, nil
			}
			return dashedLines, nil
		}
	}

	// Non-uniform / mixed array
	lines := []string{fmt.Sprintf("%s%s[%d]:", pad, prefix, len(arr))}
	for _, item := range arr {
		subLines, err := encodeListItem(item, indent+1)
		if err != nil {
			return nil, err
		}
		lines = append(lines, subLines...)
	}
	return lines, nil
}

func encodeListItem(item Value, indent int) ([]string, error) {
	pad := strings.Repeat("  ", indent)

	if isScalar(item) {
		s, err := encodePrimitive(item)
		if err != nil {
			return nil, err
		}
		if str, ok := item.(string); ok {
			s = encodeScalar(str, false)
		}
		return []string{fmt.Sprintf("%s- %s", pad, s)}, nil
	}

	if arr, ok := item.([]Value); ok {
		subLines, err := encodeArray(arr, "", 0)
		if err != nil {
			return nil, err
		}
		lines := []string{fmt.Sprintf("%s- %s", pad, subLines[0])}
		for i := 1; i < len(subLines); i++ {
			lines = append(lines, fmt.Sprintf("%s  %s", pad, subLines[i]))
		}
		return lines, nil
	}

	obj, ok := item.(map[string]Value)
	if !ok {
		return nil, fmt.Errorf("lean: unsupported list item type %T", item)
	}

	entries := make([][2]string, 0, len(obj))
	for k, v := range obj {
		entries = append(entries, [2]string{k, ""})
		_ = v
	}
	if len(entries) == 0 {
		return []string{fmt.Sprintf("%s- {}", pad)}, nil
	}

	// Use sorted keys for deterministic output
	var keys []string
	for k := range obj {
		keys = append(keys, k)
	}

	firstKey := keys[0]
	firstVal := obj[firstKey]

	var lines []string
	if isScalar(firstVal) {
		sv, err := encodePrimitive(firstVal)
		if err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s- %s:%s", pad, firstKey, sv))
	} else if arr, ok := firstVal.([]Value); ok {
		subLines, err := encodeArray(arr, firstKey, 0)
		if err != nil {
			return nil, err
		}
		lines = append(lines, fmt.Sprintf("%s- %s", pad, subLines[0]))
		for i := 1; i < len(subLines); i++ {
			lines = append(lines, fmt.Sprintf("%s  %s", pad, subLines[i]))
		}
	} else {
		subObj, ok := firstVal.(map[string]Value)
		if !ok {
			return nil, fmt.Errorf("lean: unsupported list item value type %T", firstVal)
		}
		if len(subObj) == 0 {
			lines = append(lines, fmt.Sprintf("%s- %s:{}", pad, firstKey))
		} else {
			lines = append(lines, fmt.Sprintf("%s- %s:", pad, firstKey))
			for k, v := range subObj {
				if err := validateKey(k); err != nil {
					return nil, err
				}
				subLines, err := encodeValue(v, k, indent+2)
				if err != nil {
					return nil, err
				}
				lines = append(lines, subLines...)
			}
		}
	}

	// Remaining keys
	for i := 1; i < len(keys); i++ {
		k := keys[i]
		v := obj[k]
		if err := validateKey(k); err != nil {
			return nil, err
		}
		subLines, err := encodeValue(v, k, indent+1)
		if err != nil {
			return nil, err
		}
		lines = append(lines, subLines...)
	}

	return lines, nil
}
