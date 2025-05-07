package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const testDir = "test_output"
const testFile = "test_file.json"
const testPath = testDir + "/" + testFile

type TestData struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

/*
TestSaveToFile tests the SaveToFile utility function with:
  - JSON struct data
  - Raw string data
  - Byte array data
*/
func TestSaveToFile(t *testing.T) {
	defer os.RemoveAll(testDir)

	// Test 1: Save JSON data
	data := TestData{Name: "Brennen", Age: 30}
	err := SaveToFile(testDir, testFile, data)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(testPath); os.IsNotExist(err) {
		t.Fatalf("Expected file to exist, but it does not")
	}

	// Test 2: Save string data
	err = SaveToFile(testDir, "string_file.txt", "Hello, World!")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the file exists
	if _, err := os.Stat(filepath.Join(testDir, "string_file.txt")); os.IsNotExist(err) {
		t.Fatalf("Expected file to exist, but it does not")
	}

	// Test 3: Save byte array data
	err = SaveToFile(testDir, "bytes_file.bin", []byte{0x00, 0x01, 0x02})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

/*
TestLoadFromFile tests LoadFromFile for existing and non-existing files.
*/
func TestLoadFromFile(t *testing.T) {
	defer os.RemoveAll(testDir)

	// Setup: Save a test JSON file
	data := TestData{Name: "Brennen", Age: 30}
	SaveToFile(testDir, testFile, data)

	// Test 1: Load existing file
	content, err := LoadFromFile(testPath)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	var loadedData TestData
	err = json.Unmarshal(content, &loadedData)
	if err != nil {
		t.Fatalf("Expected valid JSON, got error: %v", err)
	}

	if loadedData.Name != "Brennen" || loadedData.Age != 30 {
		t.Fatalf("Data mismatch. Got %+v", loadedData)
	}

	// Test 2: Load non-existing file
	_, err = LoadFromFile("nonexistent.json")
	if err == nil {
		t.Fatalf("Expected an error, but got none")
	}
}

/*
TestCreateDirectoryIfNotExist tests if the directory is created if it doesn't exist.
*/
func TestCreateDirectoryIfNotExist(t *testing.T) {
	defer os.RemoveAll("new_directory")

	err := CreateDirectoryIfNotExist("new_directory")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify the directory exists
	if _, err := os.Stat("new_directory"); os.IsNotExist(err) {
		t.Fatalf("Expected directory to exist, but it does not")
	}
}

/*
TestGetTimestamp checks if GetTimestamp returns a non-empty string.
*/
func TestGetTimestamp(t *testing.T) {
	timestamp := GetTimestamp()
	if timestamp == "" {
		t.Fatalf("Expected a timestamp, got an empty string")
	}
}
