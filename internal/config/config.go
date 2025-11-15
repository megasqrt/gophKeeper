package config

// План реализации:
// 1. Определить структуру конфигурации.
// 2. Реализовать функцию для загрузки конфигурации из файла или переменных окружения.

import (
	"flag"
	"gophKeeper/internal/helper"
	"time"
	"log"
	"github.com/spf13/viper"
)

type Config struct {
	ServerAddress   string
	StoreInterval   time.Duration
	FileStoragePath string
	Restore         bool
	DatabaseDSN     string
	HashKey         string
	CryptoKey       string
	ConfigPath      string
	TrustedSubnet   string
}

var cfg Config

const (
	defaultServerAddress   = "localhost:8080"
	defaultFileStoragePath = "/tmp/metrics-db.json"
	defaultRestore         = true
	defaultDatabaseDSN     = ""
	defaultHashKey         = ""
	defaultCryptoKey       = ""
	defaultTrustedSubnet   = ""
	defaultConfigPath      = ""
)

func NewConfig() Config {

	flag.StringVar(&cfg.ServerAddress, "a", defaultServerAddress, "server address")
	flag.StringVar(&cfg.FileStoragePath, "f", defaultFileStoragePath, "file storage path")
	flag.BoolVar(&cfg.Restore, "r", defaultRestore, "restore from file on start")
	flag.StringVar(&cfg.DatabaseDSN, "d", defaultDatabaseDSN, "database DSN")
	flag.StringVar(&cfg.HashKey, "k", defaultHashKey, "key for hashing")
	flag.StringVar(&cfg.CryptoKey, "crypto-key", defaultCryptoKey, "path to private key file")
	flag.StringVar(&cfg.TrustedSubnet, "t", defaultTrustedSubnet, "trusted subnet in CIDR format")
	flag.StringVar(&cfg.ConfigPath, "c", defaultConfigPath, "path to config file")
	flag.Parse()

	// Сначала читаем конфиг из файла, если он указан
	configPathFromEnv := viper.Get("CONFIG")
	if cfg.ConfigPath != "" {
		viper.SetConfigFile(cfg.ConfigPath)
	} else if configPathFromEnv != nil {
		viper.SetConfigFile(configPathFromEnv.(string))
	}

	if viper.ConfigFileUsed() != "" {
		if err := viper.ReadInConfig(); err != nil {
			// Логируем ошибку, но не падаем, т.к. конфиг не обязателен
			//logger.NewZerologLogger()Warn().Error(err).Msg("Error reading config file")
			log.Printf("Error reading config file: %s", err)
		}
	}

	viper.AutomaticEnv()

	helper.AssignFromViperIfSet(&cfg.ServerAddress, "ADDRESS", viper.GetString, defaultServerAddress)
	helper.AssignFromViperIfSet(&cfg.FileStoragePath, "FILE_STORAGE_PATH", viper.GetString, defaultFileStoragePath)
	helper.AssignFromViperIfSet(&cfg.Restore, "RESTORE", viper.GetBool, defaultRestore)
	helper.AssignFromViperIfSet(&cfg.DatabaseDSN, "DATABASE_DSN", viper.GetString, defaultDatabaseDSN)
	helper.AssignFromViperIfSet(&cfg.HashKey, "KEY", viper.GetString, defaultHashKey)
	helper.AssignFromViperIfSet(&cfg.CryptoKey, "CRYPTO_KEY", viper.GetString, defaultCryptoKey)
	helper.AssignFromViperIfSet(&cfg.TrustedSubnet, "TRUSTED_SUBNET", viper.GetString, defaultTrustedSubnet)
	helper.AssignFromViperIfSet(&cfg.ConfigPath, "CONFIG", viper.GetString, defaultConfigPath)

	return cfg
}