package rofi

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
)

const maxHistoryCount = 5

func SortUsingHistory(opts []Option, namespace string) []Option {
	cache, err := getCachePath(namespace)
	if err != nil {
		log.Printf("Error while finding cache: %s\n", err)
		return opts
	}
	f, err := os.OpenFile(cache, os.O_RDONLY, 0666)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error while opening cache: %s\n", err)
		}
		return opts
	}
	defer f.Close()

	history, err := readHistory(f)
	if err != nil {
		log.Printf("Error while reading history: %s", err)
		return opts
	}

	prio := []Option{}

	for _, h := range history {
		for i, opt := range opts {
			if h == opt.Value {
				opts = append(opts[:i], opts[i+1:]...)
				prio = append(prio, opt)
			}
		}
	}

	return append(prio, opts...)
}

func SaveToHistory(namespace, value string) {
	isNewFile := false
	var history []string

	cache, err := getCachePath(namespace)
	if err != nil {
		log.Printf("Error while finding cache: %s\n", err)
		return
	}

	if err := os.MkdirAll(path.Dir(cache), os.ModePerm); err != nil {
		log.Printf("Error while creating path: %s\n", err)
		return
	}

	_, err = os.Stat(cache)
	if err != nil && os.IsNotExist(err) {
		isNewFile = true
	}

	f, err := os.OpenFile(cache, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Printf("Error while opening cache: %s\n", err)
		return
	}

	defer f.Close()

	if !isNewFile {
		history, err = readHistory(f)
		if err != nil {
			log.Printf("Error while reading history: %s\n", err)
		}

		// shifting
		for i, val := range history {
			if val == value {
				history = append(history[:i], history[i+1:]...)
			}
		}
	}

	nextHistory := []string{value}
	if len(history) >= maxHistoryCount {
		nextHistory = append(nextHistory, history[:maxHistoryCount-1]...)
	} else {
		nextHistory = append(nextHistory, history...)
	}

	if err := writeHistory(f, nextHistory); err != nil {
		log.Printf("Error while saving history: %s\n", err)
	}
}

func getCachePath(namespace string) (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return path.Join(cache, "/rofi/", namespace+".json"), nil
}

func readHistory(f *os.File) ([]string, error) {
	var content []string

	b, err := io.ReadAll(f)
	if err != nil {
		return content, err
	}

	err = json.Unmarshal(b, &content)

	return content, err
}

func writeHistory(f *os.File, content []string) error {
	b, err := json.MarshalIndent(content, "", "  ")

	if err != nil {
		return fmt.Errorf("error while marshalling history: %w", err)
	}

	if err := f.Truncate(0); err != nil {
		return fmt.Errorf("error while clearing history: %w", err)
	}

	if _, err := f.WriteAt(b, 0); err != nil {
		return fmt.Errorf("error while writing history: %w", err)
	}

	return nil
}
