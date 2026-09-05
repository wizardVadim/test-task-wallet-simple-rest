package app

import (
	"fmt"
	"wallet-app/internal/core/config"
)

func Run() error {
	config, err := config.Load()
	if err != nil {
		return err
	}
	_ = config

	fmt.Println("app has been started")
	return nil
}
