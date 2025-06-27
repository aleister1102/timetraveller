package main

import (
	"bufio"
	"os"
)

func writeUrlsToFile(filename string, urls []string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, url := range urls {
		if _, err := writer.WriteString(url + "\n"); err != nil {
			return err
		}
	}
	return writer.Flush()
}
