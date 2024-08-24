package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
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
	// Create a temporary directory for the cache
	tempDir, err := os.MkdirTemp("", "aocgen_cache")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a challenges file in the temporary cache directory
	challengesFile := filepath.Join(tempDir, "challenges.json")
	testData := []Challenge{
		{Name: "day1_part1_2015", Input: "test input", Answer: "280", Task: "test task"},
	}
	data, _ := json.Marshal(testData)
	err = os.WriteFile(challengesFile, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	challenges, err := loadChallenges(tempDir, "challenges.json")
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

	correct, err := evaluateSolution(challenge, tmpfile.Name(), "python", 5*time.Second)
	if err != nil {
		t.Fatalf("Failed to evaluate solution: %v", err)
	}

	if !correct {
		t.Errorf("Solution evaluation failed, expected correct solution")
	}

	// Test incorrect solution
	challenge.Answer = "24"
	correct, err = evaluateSolution(challenge, tmpfile.Name(), "python", 5*time.Second)
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

func TestDownloadChallenge(t *testing.T) {
	// Set up a mock server to simulate Advent of Code website
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session")
		if err != nil || sessionCookie.Value != "test_session" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		switch r.URL.Path {
		case "/2022/day/1":
			w.Write([]byte(`<article class="day-desc">
                <h2>--- Day 1: Calorie Counting ---</h2>
                <p>Santa's reindeer typically eat regular reindeer food, but they need a lot of magical energy to deliver presents on Christmas.</p>
                <h2>--- Part Two ---</h2>
                <p>By the time you calculate the answer to the Elves' question, they've already realized that the Elf carrying the most Calories of food might eventually run out of snacks.</p>
            </article>`))
		case "/2022/day/1/input":
			w.Write([]byte("3120\n4127\n1830\n1283\n5021\n3569"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Replace the actual URL with our test server URL
	originalAocBaseURL := aocBaseURL
	aocBaseURL = server.URL
	defer func() { aocBaseURL = originalAocBaseURL }()

	testCases := []struct {
		name            string
		part            int
		expectedName    string
		expectedTitle   string
		expectedContent []string
	}{
		{
			name:            "Part 1",
			part:            1,
			expectedName:    "day1_part1_2022",
			expectedTitle:   "--- Day 1: Calorie Counting ---",
			expectedContent: []string{"Santa's reindeer typically eat regular reindeer food"},
		},
		{
			name:          "Part 2",
			part:          2,
			expectedName:  "day1_part2_2022",
			expectedTitle: "--- Day 1: Calorie Counting ---",
			expectedContent: []string{
				"Santa's reindeer typically eat regular reindeer food",
				"--- Part Two ---",
				"By the time you calculate the answer to the Elves' question",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flags := Flags{
				Day:     1,
				Year:    2022,
				Part:    tc.part,
				Session: "test_session",
			}

			err := downloadChallenge(flags)
			if err != nil {
				t.Fatalf("Failed to download challenge: %v", err)
			}

			challenges, err := loadChallenges(cacheDir, "challenges.json")
			if err != nil {
				t.Fatalf("Failed to load challenges: %v", err)
			}

			if len(challenges) == 0 {
				t.Fatalf("No challenges loaded")
			}

			challenge := challenges[len(challenges)-1]

			if challenge.Name != tc.expectedName {
				t.Errorf("Expected challenge name %s, got %s", tc.expectedName, challenge.Name)
			}

			// Print out the actual task content
			t.Logf("Actual task content for %s:\n%s", tc.name, challenge.Task)

			if !strings.Contains(challenge.Task, tc.expectedTitle) {
				t.Errorf("Challenge task does not contain expected title.\nExpected: %s\nGot: %s", tc.expectedTitle, challenge.Task)
			}

			for _, content := range tc.expectedContent {
				if !strings.Contains(challenge.Task, content) {
					t.Errorf("Challenge task does not contain expected content.\nExpected to find: %s\nIn: %s", content, challenge.Task)
				}
			}

			expectedInput := "3120\n4127\n1830\n1283\n5021\n3569"
			if challenge.Input != expectedInput {
				t.Errorf("Challenge input does not match expected content. Got: %s, Want: %s", challenge.Input, expectedInput)
			}

			if challenge.Answer != "" {
				t.Errorf("Expected empty answer for new challenge, got: %s", challenge.Answer)
			}
		})
	}
}

func TestDownloadChallengeWithAnswers(t *testing.T) {
	testCases := []struct {
		name            string
		part            int
		responseBody    string
		expectedTitle   string
		expectedContent string
		unexpectedText  string
	}{
		{
			name: "Part 1 with answer",
			part: 1,
			responseBody: `<article class="day-desc">
                <h2>--- Day 1: Calorie Counting ---</h2>
                <p>Santa's reindeer typically eat regular reindeer food, but they need a lot of magical energy to deliver presents on Christmas.</p>
                <p>Your puzzle answer was 12345.</p>
            </article>`,
			expectedTitle:   "--- Day 1: Calorie Counting ---",
			expectedContent: "Santa's reindeer typically eat regular reindeer food",
			unexpectedText:  "Your puzzle answer was",
		},
		{
			name: "Part 2 with answers",
			part: 2,
			responseBody: `<article class="day-desc">
                <h2>--- Day 1: Calorie Counting ---</h2>
                <p>Santa's reindeer typically eat regular reindeer food, but they need a lot of magical energy to deliver presents on Christmas.</p>
                <p>Your puzzle answer was 12345.</p>
                <h2 id="part2">--- Part Two ---</h2>
                <p>Now, you're ready to find the real Calorie Counting winner: the Elf carrying the most Calories.</p>
                <p>Your puzzle answer was 67890.</p>
            </article>`,
			expectedTitle:   "--- Day 1: Calorie Counting ---",
			expectedContent: "Santa's reindeer typically eat regular reindeer food",
			unexpectedText:  "Your puzzle answer was",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			originalAocBaseURL := aocBaseURL
			aocBaseURL = server.URL
			defer func() { aocBaseURL = originalAocBaseURL }()

			flags := Flags{
				Day:     1,
				Year:    2023,
				Part:    tc.part,
				Session: "test_session",
			}

			err := downloadChallenge(flags)
			if err != nil {
				t.Fatalf("Failed to download challenge: %v", err)
			}

			challenges, err := loadChallenges(cacheDir, "challenges.json")
			if err != nil {
				t.Fatalf("Failed to load challenges: %v", err)
			}

			if len(challenges) == 0 {
				t.Fatalf("No challenges loaded")
			}

			challenge := challenges[len(challenges)-1]

			if !strings.Contains(challenge.Task, tc.expectedTitle) {
				t.Errorf("Expected task to contain title: %q, but it doesn't", tc.expectedTitle)
			}

			if !strings.Contains(challenge.Task, tc.expectedContent) {
				t.Errorf("Expected task to contain: %q, but it doesn't", tc.expectedContent)
			}

			if strings.Contains(challenge.Task, tc.unexpectedText) {
				t.Errorf("Task should not contain: %q, but it does", tc.unexpectedText)
			}

			if tc.part == 2 {
				if !strings.Contains(challenge.Task, "--- Part Two ---") {
					t.Errorf("Expected task to contain '--- Part Two ---' for Part 2, but it doesn't")
				}
			}
		})
	}
}

func TestRealDownloadChallenge(t *testing.T) {
	if os.Getenv("RUN_REAL_DOWNLOAD_TEST") != "true" {
		t.Skip("Skipping real download test. Set RUN_REAL_DOWNLOAD_TEST=true to run this test.")
	}

	err := godotenv.Load()
	if err != nil {
		t.Fatalf("Error loading .env file: %v", err)
	}

	session := os.Getenv("ADVENT_OF_CODE_SESSION")
	if session == "" {
		t.Fatal("ADVENT_OF_CODE_SESSION not set in .env file")
	}

	testCases := []struct {
		name         string
		part         int
		expectedFile string
	}{
		{
			name:         "Download Part 1",
			part:         1,
			expectedFile: "day1_part1_2023.txt",
		},
		{
			name:         "Download Part 2",
			part:         2,
			expectedFile: "day1_part2_2023.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			flags := Flags{
				Day:     1,
				Year:    2023,
				Part:    tc.part,
				Session: session,
			}

			err := downloadChallenge(flags)
			if err != nil {
				t.Fatalf("Failed to download challenge: %v", err)
			}

			// Load the challenge from the file to check its contents
			challenges, err := loadChallenges("aocgen_cache", "challenges.json")
			if err != nil {
				t.Fatalf("Failed to load challenges: %v", err)
			}

			if len(challenges) == 0 {
				t.Fatalf("No challenges loaded")
			}

			challenge := challenges[0]

			if !strings.Contains(challenge.Task, "--- Day 1: Trebuchet?! ---") {
				t.Errorf("Challenge task does not contain expected content")
			}

			if strings.Contains(challenge.Task, "Your puzzle answer was") {
				t.Errorf("Challenge task should not contain answer")
			}

			if tc.part == 2 && !strings.Contains(challenge.Task, "--- Part Two ---") {
				t.Errorf("Part 2 challenge should contain Part Two section")
			}

			err = os.WriteFile(tc.expectedFile, []byte(challenge.Task+"\n\nInput:\n"+challenge.Input), 0644)
			if err != nil {
				t.Fatalf("Failed to write challenge to file: %v", err)
			}

			t.Logf("Successfully downloaded and saved %s", tc.expectedFile)
		})
	}
}
