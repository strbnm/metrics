package main

import (
	"flag"
	"fmt"
	"strconv"
	"time"
)

// неэкспортированная переменная flagRunAddr содержит адрес и порт для запуска сервера
var flagRunAddr string

// Значения по умолчанию для переменных flagPollInterval и flagReportInterval, если аргументов -p и/или -r не будет в команде
var flagPollInterval = 2 * time.Second
var flagReportInterval = 10 * time.Second

// parseFlags обрабатывает аргументы командной строки
// и сохраняет их значения в соответствующих переменных
func parseFlags() {
	// регистрируем переменную flagRunAddr
	// как аргумент -a со значением localhost:8080 по умолчанию
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")

	// регистрируем функцию-обработчик для аргумента -p (значение для flagPollInterval)
	flag.Func("p", "time interval between polls in seconds", func(flagValue string) error {
		seconds, err := getSeconds(flagValue)
		if err != nil {
			return err
		}
		flagPollInterval = time.Duration(seconds) * time.Second
		return nil
	})

	// регистрируем функцию-обработчик для аргумента -r (значение для flagReportInterval)
	flag.Func("r", "time interval between reports in seconds", func(flagValue string) error {
		seconds, err := getSeconds(flagValue)
		if err != nil {
			return err
		}
		flagReportInterval = time.Duration(seconds) * time.Second
		return nil
	})

	// парсим переданные серверу аргументы в зарегистрированные переменные
	flag.Parse()
}

func getSeconds(flagValue string) (int64, error) {
	seconds, err := strconv.ParseInt(flagValue, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number format: %v", err)
	}
	if seconds < 0 {
		return 0, fmt.Errorf("duration cannot be negative")
	}
	return seconds, nil
}
