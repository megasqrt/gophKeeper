package main

import (
	"fmt"
	"gophKeeper/client/internal/config"
	"gophKeeper/client/internal/delivery/tui"
	"gophKeeper/client/internal/storage"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Инициализируем конфигурацию.
	// Это создаст ~/.gophkeeper/gpk.conf, если его нет.
	cfg, err := config.Init()
	if err != nil {
		log.Fatalf("Failed to initialize config: %v", err)
	}

	// Инициализируем хранилище.
	store, err := storage.NewBboltStorage(cfg.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Используем новую корневую модель
	rootModel := tui.NewRootModel(store, cfg)

	p := tea.NewProgram(rootModel)
	if _, err := p.Run(); err != nil {
		log.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}
	fmt.Println("See you later!")
}
