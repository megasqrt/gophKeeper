package grpchelper

import (
	"fmt"
	"strconv"
)

type OpType string

const (
    Create  OpType = "create"
    Update  OpType = "update"
    Delete  OpType = "delete"
    //Restore OpType = "restore"
)

// SyncInfo представляет минимальный набор данных для синхронизации.
type SyncInfo struct {
	LocalID  string `json:"id"`
	ServerID string `json:"server_id"`
	Checksum string `json:"checksum"`
	ChangeTime int64 `json:"change_time"`
	SyncTime int64 `json:"sync_time"`
	OperationType  OpType `json:"operation_type"`

}

// ShortSyncResult содержит результат краткой синхронизации
type ShortSyncResult struct {
	LocalIDs  []string // ID элементов для отправки на сервер
	ServerIDs []string // ID элементов для получения с сервера
	DeletedIDs []string // ID элементов для удаления
}

// InterfaceToString safely converts an interface{} to a string.
func InterfaceToString(v interface{}) string {
	if v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
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

// FormatFileSize форматирует размер файла в читаемый вид (KB, MB, GB и т.д.).
func FormatFileSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}
