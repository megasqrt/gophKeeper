package storage

import "fmt"

// StringMapToInterfaceMap конвертирует map[string]string в map[string]interface{}.
func StringMapToInterfaceMap(m map[string]string) map[string]interface{} {
	newMap := make(map[string]interface{}, len(m))
	for k, v := range m {
		newMap[k] = v
	}
	return newMap
}

// InterfaceMapToStringMap конвертирует map[string]interface{} в map[string]string.
func InterfaceMapToStringMap(m map[string]interface{}) map[string]string {
	newMap := make(map[string]string, len(m))
	for k, v := range m {
		newMap[k] = fmt.Sprintf("%v", v)
	}
	return newMap
}
