package gotoon

import (
	"math"
	"reflect"
	"strings"
	"time"
)

// normalizeValue converts any Go value to a JSON-compatible value
func normalizeValue(value interface{}) interface{} {
	if value == nil {
		return nil
	}

	v := reflect.ValueOf(value)

	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint())

	case reflect.Float32, reflect.Float64:
		f := v.Float()
		// Handle special float values
		if math.IsNaN(f) || math.IsInf(f, 0) {
			return nil
		}
		// Normalize -0 to 0
		if f == 0 {
			return 0.0
		}
		return f

	case reflect.String:
		return v.String()

	case reflect.Slice, reflect.Array:
		arr := make([]interface{}, v.Len())
		for i := 0; i < v.Len(); i++ {
			arr[i] = normalizeValue(v.Index(i).Interface())
		}
		return arr

	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			// Non-string keys not supported, return null
			return nil
		}
		obj := make(map[string]interface{})
		iter := v.MapRange()
		for iter.Next() {
			key := iter.Key().String()
			obj[key] = normalizeValue(iter.Value().Interface())
		}
		return obj

	case reflect.Struct:
		// Handle time.Time specially
		if t, ok := value.(time.Time); ok {
			return t.Format(time.RFC3339Nano)
		}

		// Convert struct to map using exported fields
		obj := make(map[string]interface{})
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			field := t.Field(i)
			// Only include exported fields
			if field.PkgPath == "" {
				fieldValue := v.Field(i)
				if fieldValue.CanInterface() {
					// Use json tag if available, otherwise use field name
					name := field.Name
					if tag := field.Tag.Get("json"); tag != "" && tag != "-" {
						// Parse the tag to extract just the field name (before the first comma)
						// This handles modifiers like ",omitempty", ",inline", etc.
						if commaIdx := strings.Index(tag, ","); commaIdx != -1 {
							name = tag[:commaIdx]
						} else {
							name = tag
						}
						// Handle empty name (like ",inline") - skip this field and inline its contents
						if name == "" {
							// For inline fields, merge their contents into parent object
							if fieldValue.Kind() == reflect.Struct ||
								(fieldValue.Kind() == reflect.Ptr && !fieldValue.IsNil() && fieldValue.Elem().Kind() == reflect.Struct) {
								inlined := normalizeValue(fieldValue.Interface())
								if inlinedMap, ok := inlined.(map[string]interface{}); ok {
									for k, v := range inlinedMap {
										obj[k] = v
									}
								}
								continue
							}
							// If not a struct, use the original field name
							name = field.Name
						}
					}
					obj[name] = normalizeValue(fieldValue.Interface())
				}
			}
		}
		return obj

	case reflect.Ptr, reflect.Interface:
		if v.IsNil() {
			return nil
		}
		return normalizeValue(v.Elem().Interface())

	default:
		// Unsupported types (func, chan, etc.) become null
		return nil
	}
}

// Type guard functions

// isPrimitive checks if a value is a JSON primitive (string, number, bool, null)
func isPrimitive(value interface{}) bool {
	if value == nil {
		return true
	}
	switch value.(type) {
	case bool, float64, string:
		return true
	default:
		return false
	}
}

// isArray checks if a value is a slice/array
func isArray(value interface{}) bool {
	if value == nil {
		return false
	}
	v := reflect.ValueOf(value)
	return v.Kind() == reflect.Slice || v.Kind() == reflect.Array
}

// isObject checks if a value is a map (after normalization)
func isObject(value interface{}) bool {
	if value == nil {
		return false
	}
	_, ok := value.(map[string]interface{})
	return ok
}

// Array type detection helpers

// isArrayOfPrimitives checks if all elements are primitives
func isArrayOfPrimitives(arr []interface{}) bool {
	for _, item := range arr {
		if !isPrimitive(item) {
			return false
		}
	}
	return true
}

// isArrayOfArrays checks if all elements are arrays
func isArrayOfArrays(arr []interface{}) bool {
	for _, item := range arr {
		if !isArray(item) {
			return false
		}
	}
	return true
}

// isArrayOfObjects checks if all elements are objects
func isArrayOfObjects(arr []interface{}) bool {
	for _, item := range arr {
		if !isObject(item) {
			return false
		}
	}
	return true
}
