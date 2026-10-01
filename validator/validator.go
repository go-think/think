package validator

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-think/think/contract"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)+$`)
var alphaRegex = regexp.MustCompile(`^[a-zA-Z]+$`)
var alphaNumRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

// Validator executes validation rules against a dataset.
type Validator struct {
	data      map[string]any
	rules     map[string]string
	messages  map[string]string
	errors    map[string][]string
	validated map[string]any
	performed bool
}

// Make creates a new Validator instance.
func Make(data any, rules map[string]string, messages ...map[string]string) *Validator {
	dataMap := toDataMap(data)
	msgMap := make(map[string]string)
	if len(messages) > 0 && messages[0] != nil {
		for k, v := range messages[0] {
			msgMap[k] = v
		}
	}

	return &Validator{
		data:      dataMap,
		rules:     rules,
		messages:  msgMap,
		errors:    make(map[string][]string),
		validated: make(map[string]any),
	}
}

// Validate validates data and returns validated values, or *contract.ValidationException on failure.
func Validate(data any, rules map[string]string, messages ...map[string]string) (map[string]any, error) {
	v := Make(data, rules, messages...)
	if v.Fails() {
		return nil, contract.NewValidationException(v.Errors())
	}
	return v.Validated(), nil
}

// Passes runs validation and reports whether all rules passed.
func (v *Validator) Passes() bool {
	if !v.performed {
		v.validate()
	}
	return len(v.errors) == 0
}

// Fails runs validation and reports whether any rules failed.
func (v *Validator) Fails() bool {
	return !v.Passes()
}

// Errors returns all validation errors.
func (v *Validator) Errors() map[string][]string {
	if !v.performed {
		v.validate()
	}
	return v.errors
}

// Validated returns the map of validated fields that passed validation.
func (v *Validator) Validated() map[string]any {
	if !v.performed {
		v.validate()
	}
	return v.validated
}

func (v *Validator) validate() {
	v.performed = true
	v.errors = make(map[string][]string)
	v.validated = make(map[string]any)

	for field, ruleLine := range v.rules {
		val, exists := v.data[field]
		rules := strings.Split(ruleLine, "|")

		hasRequired := false
		for _, r := range rules {
			if strings.TrimSpace(r) == "required" {
				hasRequired = true
				break
			}
		}

		// If field is missing or empty and not required, skip other validations
		if !hasRequired && (!exists || isEmpty(val)) {
			continue
		}

		fieldFailed := false
		for _, rule := range rules {
			rule = strings.TrimSpace(rule)
			if rule == "" {
				continue
			}

			ruleName, ruleParam := parseRule(rule)
			if !v.checkRule(field, val, ruleName, ruleParam) {
				msg := v.formatMessage(field, ruleName, ruleParam)
				v.errors[field] = append(v.errors[field], msg)
				fieldFailed = true
			}
		}

		if !fieldFailed && exists {
			v.validated[field] = val
		}
	}
}

func parseRule(rule string) (string, string) {
	if idx := strings.Index(rule, ":"); idx != -1 {
		return rule[:idx], rule[idx+1:]
	}
	return rule, ""
}

func (v *Validator) hasNumericRule(field string) bool {
	ruleLine, ok := v.rules[field]
	if !ok {
		return false
	}
	for _, r := range strings.Split(ruleLine, "|") {
		r = strings.TrimSpace(r)
		if r == "numeric" || r == "integer" {
			return true
		}
	}
	return false
}

func (v *Validator) checkRule(field string, val any, ruleName, ruleParam string) bool {
	switch ruleName {
	case "required":
		return !isEmpty(val)
	case "email":
		str, ok := toString(val)
		return ok && emailRegex.MatchString(str)
	case "numeric":
		_, ok := toFloat(val)
		return ok
	case "integer":
		_, ok := toInt(val)
		return ok
	case "min":
		minVal, err := strconv.ParseFloat(ruleParam, 64)
		if err != nil {
			return true
		}
		if v.hasNumericRule(field) {
			if num, ok := toFloat(val); ok {
				return num >= minVal
			}
		}
		if s, ok := val.(string); ok {
			return float64(len([]rune(s))) >= minVal
		}
		rv := reflect.ValueOf(val)
		if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
			return float64(rv.Len()) >= minVal
		}
		if num, ok := toFloat(val); ok {
			return num >= minVal
		}
		if str, ok := toString(val); ok {
			return float64(len([]rune(str))) >= minVal
		}
		return false
	case "max":
		maxVal, err := strconv.ParseFloat(ruleParam, 64)
		if err != nil {
			return true
		}
		if v.hasNumericRule(field) {
			if num, ok := toFloat(val); ok {
				return num <= maxVal
			}
		}
		if s, ok := val.(string); ok {
			return float64(len([]rune(s))) <= maxVal
		}
		rv := reflect.ValueOf(val)
		if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array) {
			return float64(rv.Len()) <= maxVal
		}
		if num, ok := toFloat(val); ok {
			return num <= maxVal
		}
		if str, ok := toString(val); ok {
			return float64(len([]rune(str))) <= maxVal
		}
		return false
	case "in":
		allowed := strings.Split(ruleParam, ",")
		str, ok := toString(val)
		if !ok {
			return false
		}
		for _, item := range allowed {
			if strings.TrimSpace(item) == str {
				return true
			}
		}
		return false
	case "not_in":
		disallowed := strings.Split(ruleParam, ",")
		str, ok := toString(val)
		if !ok {
			return true
		}
		for _, item := range disallowed {
			if strings.TrimSpace(item) == str {
				return false
			}
		}
		return true
	case "alpha":
		str, ok := toString(val)
		return ok && alphaRegex.MatchString(str)
	case "alpha_num", "alphanumeric":
		str, ok := toString(val)
		return ok && alphaNumRegex.MatchString(str)
	case "url":
		str, ok := toString(val)
		if !ok {
			return false
		}
		u, err := url.ParseRequestURI(str)
		return err == nil && u.Scheme != "" && u.Host != ""
	case "ip":
		str, ok := toString(val)
		return ok && net.ParseIP(str) != nil
	case "json":
		str, ok := toString(val)
		if !ok {
			return false
		}
		var js any
		return json.Unmarshal([]byte(str), &js) == nil
	case "boolean":
		if _, ok := val.(bool); ok {
			return true
		}
		str, ok := toString(val)
		if !ok {
			return false
		}
		lower := strings.ToLower(str)
		return lower == "true" || lower == "false" || lower == "1" || lower == "0"
	case "confirmed":
		str, ok := toString(val)
		if !ok {
			return false
		}
		confirmationField := field + "_confirmation"
		confirmVal, exists := v.data[confirmationField]
		if !exists {
			return false
		}
		confirmStr, ok2 := toString(confirmVal)
		return ok2 && str == confirmStr
	default:
		return true
	}
}

func (v *Validator) formatMessage(field, rule, param string) string {
	// 1. Custom message for field.rule (e.g. "email.required")
	if msg, ok := v.messages[field+"."+rule]; ok {
		return msg
	}
	// 2. Custom message for rule (e.g. "required")
	if msg, ok := v.messages[rule]; ok {
		return strings.ReplaceAll(msg, ":attribute", field)
	}

	// 3. Default messages matching Laravel format
	switch rule {
	case "required":
		return fmt.Sprintf("The %s field is required.", field)
	case "email":
		return fmt.Sprintf("The %s field must be a valid email address.", field)
	case "numeric":
		return fmt.Sprintf("The %s field must be a number.", field)
	case "integer":
		return fmt.Sprintf("The %s field must be an integer.", field)
	case "min":
		return fmt.Sprintf("The %s field must be at least %s.", field, param)
	case "max":
		return fmt.Sprintf("The %s field must not be greater than %s.", field, param)
	case "in":
		return fmt.Sprintf("The selected %s is invalid.", field)
	case "not_in":
		return fmt.Sprintf("The selected %s is invalid.", field)
	case "alpha":
		return fmt.Sprintf("The %s field must only contain letters.", field)
	case "alpha_num", "alphanumeric":
		return fmt.Sprintf("The %s field must only contain letters and numbers.", field)
	case "url":
		return fmt.Sprintf("The %s field must be a valid URL.", field)
	case "ip":
		return fmt.Sprintf("The %s field must be a valid IP address.", field)
	case "json":
		return fmt.Sprintf("The %s field must be a valid JSON string.", field)
	case "boolean":
		return fmt.Sprintf("The %s field must be true or false.", field)
	case "confirmed":
		return fmt.Sprintf("The %s field confirmation does not match.", field)
	default:
		return fmt.Sprintf("The %s field is invalid.", field)
	}
}

func toDataMap(data any) map[string]any {
	if data == nil {
		return make(map[string]any)
	}
	if m, ok := data.(map[string]any); ok {
		return m
	}
	if m, ok := data.(map[string]string); ok {
		res := make(map[string]any, len(m))
		for k, v := range m {
			res[k] = v
		}
		return res
	}

	val := reflect.ValueOf(data)
	for val.Kind() == reflect.Pointer || val.Kind() == reflect.Interface {
		if val.IsNil() {
			return make(map[string]any)
		}
		val = val.Elem()
	}

	res := make(map[string]any)
	if val.Kind() == reflect.Struct {
		populateStructToDataMap(res, val)
	}
	return res
}

func populateStructToDataMap(res map[string]any, val reflect.Value) {
	t := val.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fv := val.Field(i)

		// Recurse into embedded anonymous structs
		if f.Anonymous && fv.Kind() == reflect.Struct {
			populateStructToDataMap(res, fv)
			continue
		}

		if !f.IsExported() {
			continue
		}

		key := f.Name
		if jsonTag := f.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			parts := strings.Split(jsonTag, ",")
			key = parts[0]
		} else if formTag := f.Tag.Get("form"); formTag != "" && formTag != "-" {
			parts := strings.Split(formTag, ",")
			key = parts[0]
		}

		fieldVal := fv.Interface()
		res[key] = fieldVal
		lowerKey := strings.ToLower(key)
		if _, exists := res[lowerKey]; !exists {
			res[lowerKey] = fieldVal
		}
		snakeKey := toSnake(key)
		if _, exists := res[snakeKey]; !exists {
			res[snakeKey] = fieldVal
		}
	}
}

func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func isEmpty(val any) bool {
	if val == nil {
		return true
	}
	switch v := val.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []any:
		return len(v) == 0
	case []string:
		return len(v) == 0
	case map[string]any:
		return len(v) == 0
	case bool:
		return false
	default:
		rv := reflect.ValueOf(val)
		switch rv.Kind() {
		case reflect.Pointer, reflect.Interface:
			return rv.IsNil()
		case reflect.Slice, reflect.Map, reflect.Array:
			return rv.Len() == 0
		case reflect.String:
			return strings.TrimSpace(rv.String()) == ""
		}
		return false
	}
}

func toString(val any) (string, bool) {
	if val == nil {
		return "", false
	}
	switch v := val.(type) {
	case string:
		return v, true
	case fmt.Stringer:
		return v.String(), true
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v), true
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v), true
	case float32, float64:
		return fmt.Sprintf("%v", v), true
	case bool:
		return fmt.Sprintf("%t", v), true
	default:
		return "", false
	}
}

func toFloat(val any) (float64, bool) {
	if val == nil {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	case int32:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint64:
		return float64(v), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func toInt(val any) (int64, bool) {
	if val == nil {
		return 0, false
	}
	switch v := val.(type) {
	case int:
		return int64(v), true
	case int64:
		return v, true
	case int32:
		return int64(v), true
	case uint:
		return int64(v), true
	case uint64:
		return int64(v), true
	case float64:
		if v == float64(int64(v)) {
			return int64(v), true
		}
		return 0, false
	case string:
		i, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}
