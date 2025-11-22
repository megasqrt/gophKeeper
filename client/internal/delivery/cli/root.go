package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gophkeeper",
	Short: "GophKeeper is a client for storing and managing your private data.",
	Long: `A secure client-server application to store your logins, passwords,
binary data, and other private information safely.`,
}

// Execute запускает корневую команду. Это главная точка входа для CLI.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Здесь можно добавить глобальные флаги, например, адрес сервера.
	rootCmd.PersistentFlags().StringP("server", "s", "localhost:9090", "gRPC server address")

	// Добавляем подкоманды
	rootCmd.AddCommand(registerCmd)
}
