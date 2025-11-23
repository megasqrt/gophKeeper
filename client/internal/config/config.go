package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config хранит все конфигурационные параметры клиента.
type Config struct {
	ServerAddress string `mapstructure:"server_address"`
	DBPath        string `mapstructure:"db_path"`
	CACertPath    string `mapstructure:"ca_cert_path"`
}

// Init инициализирует viper и загружает конфигурацию.
// Он также создает директорию и файл конфигурации по умолчанию, если они не существуют.
func Init() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("cannot find user home directory: %w", err)
	}

	configDir := filepath.Join(home, ".gophkeeper")
	configName := "gpk.yaml"
	configPath := filepath.Join(configDir, configName)

	// Создаем директорию, если она не существует
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create config directory: %w", err)
	}

	// Устанавливаем значения по умолчанию
	viper.SetDefault("server_address", "localhost:9090")
	viper.SetDefault("db_path", filepath.Join(configDir, ".gophkeeper.db"))
	viper.SetDefault("ca_cert_path", filepath.Join(configDir, "certs/ca.crt"))

	viper.SetConfigFile(configPath)

	// Если файл конфигурации не существует, создаем его со значениями по умолчанию
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := viper.SafeWriteConfigAs(configPath); err != nil {
			return nil, fmt.Errorf("cannot write initial config file: %w", err)
		}
	}

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("cannot read config file: %w", err)
	}

	var cfg Config
	err = viper.Unmarshal(&cfg)
	return &cfg, err
}
