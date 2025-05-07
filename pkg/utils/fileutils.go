package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

/*
SaveToFile writes the provided data to the specified file path.
It automatically creates the directory if it does not exist.

Parameters:
  - path: The directory where the file will be saved.
  - filename: The name of the file.
  - data: The data to write, either as JSON or plain text.

Returns:
  - error: An error object if the save fails, otherwise nil.
*/
func SaveToFile(path string, filename string, data interface{}) error {
	CreateDirectoryIfNotExist(path)

	fullPath := filepath.Join(path, filename)

	var output []byte
	var err error

	switch v := data.(type) {
	case string:
		output = []byte(v)
	case []byte:
		output = v
	default:
		output, err = json.MarshalIndent(data, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal data: %w", err)
		}
	}

	err = ioutil.WriteFile(fullPath, output, 0644)
	if err != nil {
		return fmt.Errorf("failed to write to file %s: %w", fullPath, err)
	}

	return nil
}

/*
LoadFromFile reads JSON or text data from a specified file path.

Parameters:
  - path: The path to the file.

Returns:
  - []byte: The content of the file.
  - error: An error object if reading fails, otherwise nil.
*/
func LoadFromFile(path string) ([]byte, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file %s does not exist", path)
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return content, nil
}

/*
CreateDirectoryIfNotExist checks if a directory exists, and creates it if it doesn't.

Parameters:
  - path: The directory path.

Returns:
  - error: An error object if the directory cannot be created, otherwise nil.
*/
func CreateDirectoryIfNotExist(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}
	return nil
}

/*
GetTimestamp generates a formatted timestamp string.

Returns:
  - string: A formatted timestamp string for filenames.
*/
func GetTimestamp() string {
	return time.Now().Format("2006-01-02_15-04-05")
}
