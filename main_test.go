package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// TestParseFlags tests the parsing of command-line flags
func TestParseFlags(t *testing.T) {
	// Save original os.Args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Test case
	os.Args = []string{"cmd", "--day=1", "--part=1", "--year=2015", "--lang=python", "--model=ollama/llama3:8b", "--model_api=http://localhost:11434/v1/chat/completions"}

	flags, err := parseFlags()
	if err != nil {
		t.Fatalf("Failed to parse flags: %v", err)
	}

	if flags.Day != 1 || flags.Part != 1 || flags.Year != 2015 || flags.Lang != "python" ||
		flags.Model != "ollama/llama3:8b" || flags.ModelAPI != "http://localhost:11434/v1/chat/completions" {
		t.Errorf("Parsed flags do not match expected values")
	}
}

// TestLoadChallenges tests loading challenges from the JSON file
func TestLoadChallenges(t *testing.T) {
	// Create a temporary JSON file for testing
	tmpFile, err := os.CreateTemp("", "challenges*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write test data to the file
	testData := []Challenge{
		{Name: "day1_part1_2015", Input: "test input", Answer: "280", Task: "test task"},
	}
	json.NewEncoder(tmpFile).Encode(testData)
	tmpFile.Close()

	challenges, err := loadChallenges(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to load challenges: %v", err)
	}

	if len(challenges) != 1 || challenges[0].Name != "day1_part1_2015" {
		t.Errorf("Loaded challenges do not match expected data")
	}
}

func TestGenerateSolutionFile(t *testing.T) {
	challenge := Challenge{
		Name:  "day1_part1_2015",
		Input: "test input",
		Task:  "test task",
	}
	flags := Flags{
		Day:      1,
		Part:     1,
		Year:     2015,
		Lang:     "python",
		Model:    "test",
		ModelAPI: "http://example.com", // This is not used for "test" model, but included for completeness
	}

	err := generateSolutionFile(challenge, flags)
	if err != nil {
		t.Fatalf("Failed to generate solution file: %v", err)
	}

	// Check if file was created with correct extension
	filename := "day1_part1_2015.py"
	_, err = os.Stat(filename)
	if os.IsNotExist(err) {
		t.Errorf("Solution file was not created")
	} else {
		// Clean up only if file was created
		os.Remove(filename)
	}
}

func TestGenerateSolutionFileUnsupportedLang(t *testing.T) {
	challenge := Challenge{
		Name:  "day1_part1_2015",
		Input: "test input",
		Task:  "test task",
	}
	flags := Flags{
		Day:   1,
		Part:  1,
		Year:  2015,
		Lang:  "unsupported",
		Model: "test-model",
	}

	err := generateSolutionFile(challenge, flags)
	if err == nil {
		t.Errorf("Expected error for unsupported language, but got none")
	}

	// Check that no file was created
	filename := "day1_part1_2015.unsupported"
	_, err = os.Stat(filename)
	if !os.IsNotExist(err) {
		t.Errorf("File was created for unsupported language")
		// Clean up if file was unexpectedly created
		os.Remove(filename)
	}
}

// TestCreateInputFile tests the creation of an input file
func TestCreateInputFile(t *testing.T) {
	challenge := Challenge{
		Name:  "day1_part1_2015",
		Input: "test input",
	}

	err := createInputFile(challenge)
	if err != nil {
		t.Fatalf("Failed to create input file: %v", err)
	}

	// Check if file was created and contains correct content
	content, err := os.ReadFile("input.txt")
	if err != nil {
		t.Fatalf("Failed to read input file: %v", err)
	}

	if string(content) != challenge.Input {
		t.Errorf("Input file content does not match expected input")
	}

	// Clean up
	os.Remove("input.txt")
}

// TestFindChallenge tests finding a specific challenge
func TestFindChallenge(t *testing.T) {
	challenges := []Challenge{
		{Name: "day1_part1_2015", Input: "test input 1", Answer: "280", Task: "test task 1"},
		{Name: "day2_part1_2015", Input: "test input 2", Answer: "123", Task: "test task 2"},
	}

	flags := Flags{Day: 1, Part: 1, Year: 2015}
	challenge, err := findChallenge(challenges, flags)
	if err != nil {
		t.Fatalf("Failed to find challenge: %v", err)
	}

	if challenge.Name != "day1_part1_2015" {
		t.Errorf("Found incorrect challenge")
	}

	// Test for non-existent challenge
	flags = Flags{Day: 3, Part: 1, Year: 2015}
	_, err = findChallenge(challenges, flags)
	if err == nil {
		t.Errorf("Expected error for non-existent challenge, but got none")
	}
}

func TestDownloadChallenge(t *testing.T) {
	// Mock server to simulate Advent of Code website
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/2024/day/1" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("--- Day 1: Test Challenge ---\nThis is a test challenge description."))
		} else if r.URL.Path == "/2024/day/1/input" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Test input data"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Replace the actual URL with our test server URL
	originalBaseURL := aocBaseURL
	aocBaseURL = server.URL
	defer func() { aocBaseURL = originalBaseURL }()

	flags := Flags{
		Day:     1,
		Year:    2024,
		Session: "test_session_token",
	}

	challenge, err := downloadChallenge(flags)
	if err != nil {
		t.Fatalf("Failed to download challenge: %v", err)
	}

	expectedName := "day1_part1_2024"
	if challenge.Name != expectedName {
		t.Errorf("Expected challenge name %s, got %s", expectedName, challenge.Name)
	}

	if challenge.Task != "--- Day 1: Test Challenge ---\nThis is a test challenge description." {
		t.Errorf("Challenge task does not match expected content")
	}

	if challenge.Input != "Test input data" {
		t.Errorf("Challenge input does not match expected content")
	}
}

func TestEvaluateSolution(t *testing.T) {
	// Create a temporary solution file
	tmpfile, err := os.CreateTemp("", "solution*.py")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	// Write a simple solution that returns "42"
	_, err = tmpfile.Write([]byte("print(42)"))
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpfile.Close()

	challenge := Challenge{
		Name:   "day1_part1_2024",
		Answer: "42",
	}

	correct, err := evaluateSolution(tmpfile.Name(), "python", challenge)
	if err != nil {
		t.Fatalf("Failed to evaluate solution: %v", err)
	}

	if !correct {
		t.Errorf("Solution evaluation failed, expected correct solution")
	}

	// Test incorrect solution
	challenge.Answer = "24"
	correct, err = evaluateSolution(tmpfile.Name(), "python", challenge)
	if err != nil {
		t.Fatalf("Failed to evaluate solution: %v", err)
	}

	if correct {
		t.Errorf("Solution evaluation failed, expected incorrect solution")
	}
}

func TestGenerateCodeWithAI(t *testing.T) {
	challenge := Challenge{
		Name: "day1_part1_2024",
		Task: "Calculate the sum of all numbers in the input.",
	}
	flags := Flags{
		Lang:  "python",
		Model: "test",
	}

	code, err := generateCodeWithAI(challenge, flags)
	if err != nil {
		t.Fatalf("Failed to generate code with AI: %v", err)
	}

	if !strings.Contains(code, "print('Hello, World!')") {
		t.Errorf("Generated code does not match expected test output")
	}
}

func TestGenerateCodeWithAIOllama(t *testing.T) {
	// Create a mock server to simulate Ollama API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Errorf("Expected to request '/v1/chat/completions', got: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type: application/json, got: %s", r.Header.Get("Content-Type"))
		}

		var requestBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)

		if requestBody["model"] != "gemma2:2b-instruct-q8_0" {
			t.Errorf("Expected model: gemma2:2b-instruct-q8_0, got: %s", requestBody["model"])
		}

		messages, ok := requestBody["messages"].([]interface{})
		if !ok || len(messages) != 2 {
			t.Errorf("Expected 2 messages, got: %v", messages)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"response": "```python\n# Some Python code\n```",
		})
	}))
	defer server.Close()

	challenge := Challenge{
		Name: "day1_part1_2024",
		Task: "Calculate the sum of all numbers in the input.",
	}
	flags := Flags{
		Lang:     "python",
		Model:    "ollama/gemma2:2b-instruct-q8_0",
		ModelAPI: server.URL + "/v1/chat/completions",
	}

	code, err := generateCodeWithAI(challenge, flags)
	if err != nil {
		t.Fatalf("Failed to generate code with AI: %v", err)
	}

	if code == "" {
		t.Errorf("Generated code is empty")
	}

	if len(code) < 10 { // Arbitrary small number to ensure we got some content
		t.Errorf("Generated code is suspiciously short: %s", code)
	}
}
