package model

import (
	"fmt"
	"strconv"
	"time"
)

// InterfaceToString safely converts an interface{} to a string.
func InterfaceToString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

// InterfaceToTime safely converts an interface{} to a time.Time.
func InterfaceToTime(v interface{}) (time.Time, error) {
	if v == nil {
		return time.Time{}, nil
	}
	timeStr, ok := v.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("value is not a string: %T", v)
	}
	return time.Parse(time.RFC3339Nano, timeStr)
}

// InterfaceToBool safely converts an interface{} to a bool.
func InterfaceToBool(v interface{}) (bool, error) {
	if v == nil {
		return false, nil
	}
	return strconv.ParseBool(InterfaceToString(v))
}

// InterfaceToInt64 safely converts an interface{} to an int64.
func InterfaceToInt64(v interface{}) (int64, error) {
	return strconv.ParseInt(InterfaceToString(v), 10, 64)
}
