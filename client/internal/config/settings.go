package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// GetSettings читает текущие настройки из viper.
// Возвращает карту, где ключ - это название настройки, а значение - ее текущее значение.
func GetSettings() map[string]interface{} {
	return viper.AllSettings()
}

// UpdateSettings обновляет значения настроек в viper и сохраняет их в файл gpk.yaml.
// Принимает карту, где ключ - это название настройки, а значение - новое значение.
func UpdateSettings(newSettings map[string]interface{}) error {
	for key, value := range newSettings {
		// Проверяем, существует ли такая настройка, чтобы пользователь не мог добавить новую.
		if !viper.IsSet(key) {
			return fmt.Errorf("setting '%s' does not exist and cannot be added", key)
		}
		viper.Set(key, value)
	}

	// Сохраняем все изменения в конфигурационный файл.
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}
