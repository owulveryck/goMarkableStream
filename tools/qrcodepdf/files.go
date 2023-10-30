package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

func searchContentFiles(dirPath string) []string {
	matchedFiles := make([]string, 0)
	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		fmt.Println("Error reading directory:", err)
		return matchedFiles
	}

	for _, file := range files {
		if file.IsDir() {
			continue // Skip directories
		}

		if filepath.Ext(file.Name()) == ".metadata" {
			fullPath := filepath.Join(dirPath, file.Name())
			if checkFileForJSONContent(fullPath) {
				matchedFiles = append(matchedFiles, fullPath)
			}
		}
	}
	return matchedFiles
}

func checkFileForJSONContent(filePath string) bool {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("Error reading file:", err)
		return false
	}

	var jsonData map[string]interface{}
	if err := json.Unmarshal(content, &jsonData); err != nil {
		fmt.Println("Error decoding JSON in file:", filePath, err)
		return false
	}

	if visibleName, ok := jsonData["visibleName"]; ok {
		if visibleNameStr, ok := visibleName.(string); ok && visibleNameStr == qrPDFName {
			return true
		}
	}
	return false
}
