package main

import (
	"flag"
	"fmt"
	"strconv"
	"time"
)

// неэкспортированная переменная flagRunAddr содержит адрес и порт для запуска сервера
var flagRunAddr string
var flagPollInterval time.Duration = 2 * time.Second
var flagReportInterval time.Duration = 10 * time.Second

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением :8080 по умолчанию
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.Func("p", "time interval between polls in seconds", func(flagValue string) error {
		seconds, err := strconv.ParseInt(flagValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid number format: %v", err)
		}
		if seconds < 0 {
			return fmt.Errorf("duration cannot be negative")
		}
		flagPollInterval = time.Duration(seconds) * time.Second
		return nil
	})
	flag.Func("r", "time interval between reports in seconds", func(flagValue string) error {
		seconds, err := strconv.ParseInt(flagValue, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid number format: %v", err)
		}
		if seconds < 0 {
			return fmt.Errorf("duration cannot be negative")
		}
		flagPollInterval = time.Duration(seconds) * time.Second
		return nil
	})
	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()
}
