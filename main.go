package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

type Flags struct {
	Day      int
	Part     int
	Year     int
	Lang     string
	Model    string
	ModelAPI string
	Session  string
}

type Challenge struct {
	Name   string `json:"name"`
	Input  string `json:"input"`
	Answer string `json:"answer"`
	Task   string `json:"task"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var aocBaseURL = "https://adventofcode.com"

func parseFlags() (Flags, error) {
	flags := Flags{}
	flag.IntVar(&flags.Day, "day", 0, "Day of the challenge")
	flag.IntVar(&flags.Part, "part", 0, "Part of the challenge")
	flag.IntVar(&flags.Year, "year", 0, "Year of the challenge")
	flag.StringVar(&flags.Lang, "lang", "", "Programming language for the solution")
	flag.StringVar(&flags.Model, "model", "", "AI model to use")
	flag.StringVar(&flags.ModelAPI, "model_api", "", "API endpoint for the AI model")
	flag.Parse()
	return flags, nil
}

func loadChallenges(filename string) ([]Challenge, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var challenges []Challenge
	err = json.NewDecoder(file).Decode(&challenges)
	return challenges, err
}

// function to map languages to file extensions
func getFileExtension(lang string) (string, error) {
	extensions := map[string]string{
		"go":           "go",
		"python":       "py",
		"javascript":   "js",
		"java":         "java",
		"scala":        "scala",
		"kotlin":       "kt",
		"groovy":       "groovy",
		"clojure":      "clj",
		"csharp":       "cs",
		"fsharp":       "fs",
		"swift":        "swift",
		"objectivec":   "m",
		"r":            "r",
		"haskell":      "hs",
		"ocaml":        "ml",
		"racket":       "rkt",
		"scheme":       "scm",
		"ruby":         "rb",
		"erlang":       "erl",
		"elixir":       "ex",
		"rust":         "rs",
		"c":            "c",
		"cpp":          "cpp",
		"zig":          "zig",
		"fortran90":    "f90",
		"perl":         "pl",
		"pascal":       "pas",
		"crystal":      "cr",
		"julia":        "jl",
		"lua":          "lua",
		"php":          "php",
		"dart":         "dart",
		"bash":         "sh",
		"awk":          "awk",
		"nim":          "nim",
		"d":            "d",
		"v":            "v",
		"prolog":       "pl",
		"tcl":          "tcl",
		"coffeescript": "coffee",
		"typescript":   "ts",
	}
	ext, ok := extensions[lang]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s", lang)
	}
	return ext, nil
}

func generateSolutionFile(challenge Challenge, flags Flags) error {
	ext, err := getFileExtension(flags.Lang)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("day%d_part%d_%d.%s", flags.Day, flags.Part, flags.Year, ext)

	code, err := generateCodeWithAI(challenge, flags)
	if err != nil {
		return fmt.Errorf("error generating code with AI: %v", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(code)
	return err
}

func callOllamaAPI(apiURL, model string, messages []Message) (map[string]interface{}, error) {
	requestBody, err := json.Marshal(map[string]interface{}{
		"model":    model,
		"messages": messages,
	})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func generateCodeWithAI(challenge Challenge, flags Flags) (string, error) {
	if flags.Model == "test" {
		return fmt.Sprintf(`# Test model response for %s
def solve():
    with open('input.txt', 'r') as file:
        input_data = file.read()
    # TODO: Implement solution
    print('Hello, World!')

if __name__ == '__main__':
    solve()`, flags.Lang), nil
	}

	prompt := fmt.Sprintf(`Write %s program that reads input from a file called input.txt and prints the output to standard output.
Focus on writing clean, efficient code that demonstrates your programming skills by concisely solving the challenge.

Coding challenge:
%s

Respond only with the code surrounded by triple backticks and the language name, like this:
%s
# Your code here
%s`, flags.Lang, challenge.Task, "```"+flags.Lang, "```")

	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: prompt},
	}

	var result map[string]interface{}
	var err error

	if strings.HasPrefix(flags.Model, "ollama/") {
		result, err = callOllamaAPI(flags.ModelAPI, strings.TrimPrefix(flags.Model, "ollama/"), messages)
	} else {
		return "", fmt.Errorf("unsupported model provider: %s", flags.Model)
	}

	if err != nil {
		return "", err
	}

	content, ok := result["response"].(string)
	if !ok {
		return "", fmt.Errorf("content is not a string")
	}

	if content == "" {
		return "", fmt.Errorf("received empty response from API")
	}

	// Extract code from the response
	re := regexp.MustCompile("```(?:.*\n)?([\\s\\S]*?)```")
	matches := re.FindStringSubmatch(content)
	if len(matches) < 2 {
		return "", fmt.Errorf("no code found in the response")
	}

	code := strings.TrimSpace(matches[1])
	if code == "" {
		return "", fmt.Errorf("extracted code is empty")
	}

	return code, nil
}

func createInputFile(challenge Challenge) error {
	file, err := os.Create("input.txt")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(challenge.Input)
	return err
}

func findChallenge(challenges []Challenge, flags Flags) (Challenge, error) {
	name := fmt.Sprintf("day%d_part%d_%d", flags.Day, flags.Part, flags.Year)
	for _, c := range challenges {
		if c.Name == name {
			return c, nil
		}
	}
	return Challenge{}, fmt.Errorf("challenge not found: %s", name)
}

func parseGenerateFlags() (Flags, error) {
	flags := Flags{}
	flag.IntVar(&flags.Day, "day", 0, "Day of the challenge")
	flag.IntVar(&flags.Part, "part", 0, "Part of the challenge")
	flag.IntVar(&flags.Year, "year", 0, "Year of the challenge")
	flag.StringVar(&flags.Lang, "lang", "", "Programming language for the solution")
	flag.StringVar(&flags.Model, "model", "", "AI model to use")
	flag.StringVar(&flags.ModelAPI, "model_api", "", "API endpoint for the AI model")
	flag.Parse()
	return flags, nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Expected 'generate', 'download' or 'eval' subcommands")
		os.Exit(1)
	}

	downloadCmd := flag.NewFlagSet("download", flag.ExitOnError)
	downloadDay := downloadCmd.Int("day", 0, "Day of the challenge")
	downloadYear := downloadCmd.Int("year", 0, "Year of the challenge")
	downloadSession := downloadCmd.String("session", "", "Session token for Advent of Code")

	switch os.Args[1] {
	case "generate":
		flags, err := parseGenerateFlags()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
			os.Exit(1)
		}
		if err := runGenerateCommand(flags); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "download":
		downloadCmd.Parse(os.Args[2:])
		flags := Flags{
			Day:     *downloadDay,
			Year:    *downloadYear,
			Session: *downloadSession,
		}
		if err := runDownloadCommand(flags); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	case "eval":
		evalCmd := flag.NewFlagSet("eval", flag.ExitOnError)
		lang := evalCmd.String("lang", "", "Programming language of the solution")
		solution := evalCmd.String("solution", "", "Path to the solution file")
		evalCmd.Parse(os.Args[2:])

		if err := runEvalCommand(*lang, *solution); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Println("Expected 'generate', 'download' or 'eval' subcommands")
		os.Exit(1)
	}
}

func downloadChallenge(flags Flags) (Challenge, error) {
	if flags.Session == "" {
		return Challenge{}, fmt.Errorf("session token is required")
	}

	client := &http.Client{}
	challenge := Challenge{}

	// Download challenge description
	descURL := fmt.Sprintf("%s/%d/day/%d", aocBaseURL, flags.Year, flags.Day)
	descReq, err := http.NewRequest("GET", descURL, nil)
	if err != nil {
		return challenge, err
	}
	descReq.AddCookie(&http.Cookie{Name: "session", Value: flags.Session})

	descResp, err := client.Do(descReq)
	if err != nil {
		return challenge, err
	}
	defer descResp.Body.Close()

	if descResp.StatusCode != http.StatusOK {
		return challenge, fmt.Errorf("failed to download challenge description: %s", descResp.Status)
	}

	descBody, err := io.ReadAll(descResp.Body)
	if err != nil {
		return challenge, err
	}

	// Process the challenge description
	taskParts := strings.Split(string(descBody), "--- Part Two ---")
	challenge.Task = strings.TrimSpace(taskParts[0])

	// Remove the first part answer if present
	answerRegex := regexp.MustCompile(`(?m)Your puzzle answer was ([0-9a-zA-Z]+)\.\s*$`)
	challenge.Task = answerRegex.ReplaceAllString(challenge.Task, "")
	challenge.Task = strings.TrimSpace(challenge.Task)

	// If it's part 2, include the second part but remove its answer
	if flags.Part == 2 && len(taskParts) > 1 {
		secondPart := strings.TrimSpace(taskParts[1])
		secondPart = answerRegex.ReplaceAllString(secondPart, "")
		secondPart = strings.TrimSpace(secondPart)
		challenge.Task += "\n\n--- Part Two ---\n" + secondPart
	}

	// Download input
	inputURL := fmt.Sprintf("%s/%d/day/%d/input", aocBaseURL, flags.Year, flags.Day)
	inputReq, err := http.NewRequest("GET", inputURL, nil)
	if err != nil {
		return challenge, err
	}
	inputReq.AddCookie(&http.Cookie{Name: "session", Value: flags.Session})

	inputResp, err := client.Do(inputReq)
	if err != nil {
		return challenge, err
	}
	defer inputResp.Body.Close()

	if inputResp.StatusCode != http.StatusOK {
		return challenge, fmt.Errorf("failed to download challenge input: %s", inputResp.Status)
	}

	inputBody, err := io.ReadAll(inputResp.Body)
	if err != nil {
		return challenge, err
	}

	challenge.Name = fmt.Sprintf("day%d_part%d_%d", flags.Day, flags.Part, flags.Year)
	challenge.Input = string(inputBody)
	challenge.Answer = "" // Answer will be empty for newly downloaded challenges

	return challenge, nil
}

func saveChallenges(filename string, challenges []Challenge) error {
	file, err := json.MarshalIndent(challenges, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filename, file, 0644)
}

func evaluateSolution(filename string, lang string, challenge Challenge) (bool, error) {
	var cmd *exec.Cmd

	switch lang {
	case "python":
		cmd = exec.Command("python", filename)
	case "ruby":
		cmd = exec.Command("ruby", filename)
	// Add more languages as needed
	default:
		return false, fmt.Errorf("unsupported language: %s", lang)
	}

	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to run solution: %v", err)
	}

	// Trim whitespace and newlines from both the output and the expected answer
	result := strings.TrimSpace(string(output))
	expectedAnswer := strings.TrimSpace(challenge.Answer)

	return result == expectedAnswer, nil
}

func runGenerateCommand(flags Flags) error {
	challenges, err := loadChallenges("challenges.json")
	if err != nil {
		return fmt.Errorf("error loading challenges: %v", err)
	}

	challenge, err := findChallenge(challenges, flags)
	if err != nil {
		return fmt.Errorf("error finding challenge: %v", err)
	}

	err = createInputFile(challenge)
	if err != nil {
		return fmt.Errorf("error creating input file: %v", err)
	}

	err = generateSolutionFile(challenge, flags)
	if err != nil {
		return fmt.Errorf("error generating solution file: %v", err)
	}

	fmt.Println("Challenge files created successfully!")
	return nil
}

func runDownloadCommand(flags Flags) error {
	challenge, err := downloadChallenge(flags)
	if err != nil {
		return fmt.Errorf("error downloading challenge: %v", err)
	}

	// Save the challenge to the JSON file
	challenges, err := loadChallenges("challenges.json")
	if err != nil {
		challenges = []Challenge{}
	}

	challenges = append(challenges, challenge)
	err = saveChallenges("challenges.json", challenges)
	if err != nil {
		return fmt.Errorf("error saving challenge: %v", err)
	}

	fmt.Println("Challenge downloaded and saved successfully!")
	return nil
}

func runEvalCommand(lang, solutionPath string) error {
	// TODO: Load the challenge from the JSON file
	challenge := Challenge{} // Placeholder

	correct, err := evaluateSolution(solutionPath, lang, challenge)
	if err != nil {
		return fmt.Errorf("error evaluating solution: %v", err)
	}

	if correct {
		fmt.Println("Solution is correct!")
	} else {
		fmt.Println("Solution is incorrect.")
	}
	return nil
}
