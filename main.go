package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/apache/arrow/go/v12/arrow"
	"github.com/apache/arrow/go/v12/arrow/array"
	"github.com/apache/arrow/go/v12/arrow/memory"
	"github.com/apache/arrow/go/v12/parquet/file"
	"github.com/apache/arrow/go/v12/parquet/pqarrow"
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
	Name         string `json:"name"`
	Solution     string `json:"solution"`
	Input        string `json:"input"`
	Task         string `json:"task"`
	SolutionLang string `json:"solution_lang"`
	Year         int64  `json:"year"`
	Answer       string `json:"answer"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var baseCacheDir string

func init() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	baseCacheDir = filepath.Join(homeDir, ".aocgen")
}

func getCacheDir() string {
	return baseCacheDir
}

// Add this function to allow tests to set a custom cache directory
func setBaseCacheDir(dir string) {
	baseCacheDir = dir
}

const challengesFile = "challenges.json"
const datasetParquet = "dataset.parquet"
const datasetURL = "https://huggingface.co/datasets/isavita/advent-of-code/resolve/refs%2Fconvert%2Fparquet/default/train/0000.parquet"

var aocBaseURL = "https://adventofcode.com"

func parseFlags(args []string) (Flags, error) {
	flags := Flags{}
	flagSet := flag.NewFlagSet("", flag.ContinueOnError)
	flagSet.IntVar(&flags.Day, "day", 0, "Day of the challenge")
	flagSet.IntVar(&flags.Part, "part", 0, "Part of the challenge")
	flagSet.IntVar(&flags.Year, "year", 0, "Year of the challenge")
	flagSet.StringVar(&flags.Lang, "lang", "", "Programming language for the solution")
	flagSet.StringVar(&flags.Model, "model", "", "AI model to use")
	flagSet.StringVar(&flags.ModelAPI, "model_api", "", "API endpoint for the AI model")
	flagSet.StringVar(&flags.Session, "session", "", "Session token for Advent of Code")

	if len(args) == 0 {
		return flags, nil
	}

	err := flagSet.Parse(args)
	if err != nil {
		return flags, err
	}

	return flags, nil
}

func loadChallenges(cacheDir, filename string) ([]Challenge, error) {
	data, err := os.ReadFile(filepath.Join(cacheDir, filename))
	if err != nil {
		return nil, err
	}

	var challenges []Challenge
	err = json.Unmarshal(data, &challenges)
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

	filename := fmt.Sprintf("%s.%s", challenge.Name, ext)

	code, err := generateCodeWithAI(challenge, flags)
	if err != nil {
		return fmt.Errorf("error generating code with AI: %v", err)
	}

	err = os.WriteFile(filename, []byte(code), 0644)
	if err != nil {
		return fmt.Errorf("failed to write solution file: %v", err)
	}

	return nil
}

func callOllamaAPI(apiURL, model, prompt string) (string, error) {
	requestBody, err := json.Marshal(map[string]interface{}{
		"model":  model,
		"prompt": prompt,
	})
	if err != nil {
		return "", err
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	response, ok := result["response"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	return response, nil
}

func callOpenAIAPI(apiURL, model, prompt string) (string, error) {
	requestBody, err := json.Marshal(map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+os.Getenv("OPENAI_API_KEY"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal(body, &errorResponse); err != nil {
			return "", fmt.Errorf("API error: %s", resp.Status)
		}
		return "", fmt.Errorf("API error: %s (%s)", errorResponse.Error.Message, errorResponse.Error.Type)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}

	choices, ok := result["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		return "", fmt.Errorf("unexpected response format")
	}

	firstChoice, ok := choices[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	message, ok := firstChoice["message"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	content, ok := message["content"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	return content, nil
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

	prompt := fmt.Sprintf("Write a %s program that solves the following coding challenge:\n\n%s\n\nThe program should read input from a file called 'input.txt' and print the output to standard output.\n\nRespond ONLY with the code surrounded by triple backticks and the language name, like this:\n```%s\n<YOUR CODE HERE>\n```\nDo not include any explanations or comments outside the code block.", flags.Lang, challenge.Task, flags.Lang)

	var result string
	var err error

	switch {
	case strings.HasPrefix(flags.Model, "gpt-"):
		result, err = callOpenAIAPI(flags.ModelAPI, flags.Model, prompt)
	case strings.HasPrefix(flags.Model, "ollama/"):
		messages := []map[string]string{
			{"role": "system", "content": "You are a helpful AI assistant that generates code solutions."},
			{"role": "user", "content": prompt},
		}

		requestBody := map[string]interface{}{
			"model":    strings.TrimPrefix(flags.Model, "ollama/"),
			"messages": messages,
		}

		requestBodyBytes, err := json.Marshal(requestBody)
		if err != nil {
			return "", err
		}

		resp, err := http.Post(flags.ModelAPI, "application/json", bytes.NewBuffer(requestBodyBytes))
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		var response map[string]interface{}
		err = json.Unmarshal(body, &response)
		if err != nil {
			return "", fmt.Errorf("error unmarshaling response: %v", err)
		}

		var content string

		// Check for the simple response format
		if simpleResponse, ok := response["response"].(string); ok {
			content = simpleResponse
		} else {
			// Check for the complex response format
			choices, ok := response["choices"].([]interface{})
			if !ok || len(choices) == 0 {
				return "", fmt.Errorf("unexpected response format: 'choices' field not found or empty")
			}

			firstChoice, ok := choices[0].(map[string]interface{})
			if !ok {
				return "", fmt.Errorf("unexpected response format: first choice is not a map")
			}

			message, ok := firstChoice["message"].(map[string]interface{})
			if !ok {
				return "", fmt.Errorf("unexpected response format: 'message' field not found in first choice")
			}

			content, ok = message["content"].(string)
			if !ok {
				return "", fmt.Errorf("unexpected response format: 'content' field not found or not a string")
			}
		}

		// Extract code from the content
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
	default:
		return "", fmt.Errorf("unsupported model provider: %s", flags.Model)
	}

	if err != nil {
		return "", err
	}

	// Extract code from the result
	re := regexp.MustCompile("```(?:.*\n)?([\\s\\S]*?)```")
	matches := re.FindStringSubmatch(result)
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

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Expected 'generate', 'download', 'eval', 'list', or 'setup' subcommands")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		if err := ListChallenges(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "generate":
		flags, err := parseFlags(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
			os.Exit(1)
		}
		if err := runGenerateCommand(flags); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "download":
		flags, err := parseFlags(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
			os.Exit(1)
		}
		if err := runDownloadCommand(flags); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "eval":
		flags, err := parseFlags(os.Args[2:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
			os.Exit(1)
		}
		if err := runEvaluationCommand(flags); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "setup":
		if err := setupDataset(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Expected 'generate', 'download', 'eval', 'list', or 'setup' subcommands")
		os.Exit(1)
	}
}

func runDownloadCommand(flags Flags) error {
	return downloadChallenge(flags)
}

func downloadChallenge(flags Flags) error {
	if flags.Session == "" {
		return fmt.Errorf("session token is required")
	}

	// Set default part to 1 if not specified
	if flags.Part == 0 {
		flags.Part = 1
	}

	client := &http.Client{}
	challenge := Challenge{}

	// Download challenge description
	descURL := fmt.Sprintf("%s/%d/day/%d", aocBaseURL, flags.Year, flags.Day)
	descReq, err := http.NewRequest("GET", descURL, nil)
	if err != nil {
		return err
	}
	descReq.AddCookie(&http.Cookie{Name: "session", Value: flags.Session})

	descResp, err := client.Do(descReq)
	if err != nil {
		return err
	}
	defer descResp.Body.Close()

	if descResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download challenge description: %s", descResp.Status)
	}

	descBody, err := io.ReadAll(descResp.Body)
	if err != nil {
		return err
	}

	// Process the challenge description
	taskPartOne, taskPartTwo := cleanTaskDescription(string(descBody))

	// Download input
	inputURL := fmt.Sprintf("%s/%d/day/%d/input", aocBaseURL, flags.Year, flags.Day)
	inputReq, err := http.NewRequest("GET", inputURL, nil)
	if err != nil {
		return err
	}
	inputReq.AddCookie(&http.Cookie{Name: "session", Value: flags.Session})

	inputResp, err := client.Do(inputReq)
	if err != nil {
		return err
	}
	defer inputResp.Body.Close()

	if inputResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download challenge input: %s", inputResp.Status)
	}

	inputBody, err := io.ReadAll(inputResp.Body)
	if err != nil {
		return err
	}

	// Combine Part 1 and Part 2 for the task field if it's Part 2
	task := taskPartOne
	if flags.Part == 2 {
		task = taskPartOne + "\n\n" + taskPartTwo
	}

	challenge = Challenge{
		Name:         fmt.Sprintf("day%d_part%d_%d", flags.Day, flags.Part, flags.Year),
		Solution:     "",
		Input:        string(inputBody),
		Task:         task,
		SolutionLang: "",
		Year:         int64(flags.Year),
		Answer:       "",
	}

	// Ensure the cache directory exists
	cacheDir := getCacheDir()
	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %v", err)
	}

	// Save the challenge to the JSON file
	challenges, err := loadChallenges(cacheDir, "challenges.json")
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error loading challenges: %v", err)
	}

	challenges = append(challenges, challenge)
	err = saveChallenges("challenges.json", challenges)
	if err != nil {
		return fmt.Errorf("error saving challenge: %v", err)
	}

	fmt.Println("Challenge downloaded and saved successfully!")
	return nil
}

func cleanTaskDescription(htmlContent string) (string, string) {
	re := regexp.MustCompile(`(?s)<article class="day-desc">(.*?)</article>`)
	matches := re.FindAllStringSubmatch(htmlContent, -1)

	var partOne, partTwo string

	if len(matches) > 0 && len(matches[0]) > 1 {
		fullContent := stripTags(matches[0][1])
		fullContent = html.UnescapeString(fullContent)

		// Remove "Your puzzle answer was" and everything after it
		fullContent = regexp.MustCompile(`Your puzzle answer was.*`).ReplaceAllString(fullContent, "")

		parts := strings.Split(fullContent, "--- Part Two ---")

		partOne = strings.TrimSpace(parts[0])
		// Add a newline after the title (after the second ---)
		partOne = regexp.MustCompile(`(--- .* ---)(.*)`).ReplaceAllString(partOne, "$1\n$2")

		if len(parts) > 1 {
			partTwo = "--- Part Two ---\n" + strings.TrimSpace(parts[1])
		}
	}

	return partOne, partTwo
}

func stripTags(htmlContent string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(htmlContent, "")
}

func saveChallenges(filename string, challenges []Challenge) error {
	data, err := json.MarshalIndent(challenges, "", "  ")
	if err != nil {
		return err
	}

	cacheDir := getCacheDir()
	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create cache directory: %v", err)
	}

	return os.WriteFile(filepath.Join(cacheDir, filename), data, 0644)
}

func runGenerateCommand(flags Flags) error {
	return generateSolution(flags)
}

func generateSolution(flags Flags) error {
	challengeName := fmt.Sprintf("day%d_part%d_%d", flags.Day, flags.Part, flags.Year)
	challenges, err := loadChallenges(getCacheDir(), "challenges.json")
	if err != nil {
		return fmt.Errorf("error loading challenges: %v", err)
	}

	var challenge *Challenge
	for i, c := range challenges {
		if c.Name == challengeName {
			challenge = &challenges[i]
			break
		}
	}

	if challenge == nil {
		return fmt.Errorf("challenge not found: %s", challengeName)
	}

	err = createInputFile(*challenge)
	if err != nil {
		return fmt.Errorf("error creating input file: %v", err)
	}

	err = generateSolutionFile(*challenge, flags)
	if err != nil {
		return fmt.Errorf("error generating solution file: %v", err)
	}

	fmt.Println("Challenge files created successfully!")
	return nil
}

func runEvaluationCommand(flags Flags) error {
	challenges, err := loadChallenges(getCacheDir(), "challenges.json")
	if err != nil {
		return fmt.Errorf("error loading challenges: %v", err)
	}

	challenge, err := findChallenge(challenges, flags)
	if err != nil {
		return fmt.Errorf("error finding challenge: %v", err)
	}

	ext, err := getFileExtension(flags.Lang)
	if err != nil {
		return fmt.Errorf("error getting file extension: %v", err)
	}

	solutionPath := fmt.Sprintf("day%d_part%d_%d.%s", flags.Day, flags.Part, flags.Year, ext)

	correct, output, err := evaluateSolution(challenge, solutionPath, flags.Lang, 20*time.Second)
	if err != nil {
		return fmt.Errorf("error evaluating solution: %v", err)
	}

	if correct {
		fmt.Printf("Solution is correct!\nOutput: %s\n", output)
	} else {
		fmt.Printf("Solution is incorrect.\nOutput: %s\n", output)
	}

	return nil
}

func evaluateSolution(challenge Challenge, filename string, lang string, timeout time.Duration) (bool, string, error) {
	cmd := getCommand(lang, filename)
	if cmd == nil {
		return false, "", fmt.Errorf("unsupported language: %s", lang)
	}

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Start()
	if err != nil {
		return false, "", fmt.Errorf("failed to start command: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(timeout):
		if err := cmd.Process.Kill(); err != nil {
			return false, "", fmt.Errorf("failed to kill process: %v", err)
		}
		return false, "", fmt.Errorf("process killed as timeout reached")
	case err := <-done:
		if err != nil {
			return false, out.String(), fmt.Errorf("process finished with error: %v", err)
		}
	}

	output := out.String()
	return strings.Contains(output, challenge.Answer), output, nil
}

func getCommand(lang, filename string) *exec.Cmd {
	switch lang {
	case "python":
		return exec.Command("python", filename)
	case "javascript":
		return exec.Command("node", filename)
	case "ruby":
		return exec.Command("ruby", filename)
	case "go":
		return exec.Command("go", "run", filename)
	case "java":
		return exec.Command("java", filename)
	case "elixir":
		return exec.Command("elixir", filename)
	// Add more cases for other languages as needed
	default:
		return nil
	}
}

func ListChallenges() error {
	challenges, err := loadChallenges(getCacheDir(), "challenges.json")
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No challenges found. Use the 'download' command to get some challenges.")
			return nil
		}
		return fmt.Errorf("error loading challenges: %v", err)
	}

	if len(challenges) == 0 {
		fmt.Println("No challenges found. Use the 'download' command to get some challenges.")
		return nil
	}

	// Create a map to store challenges with their languages
	challengeMap := make(map[string][]string)

	for _, challenge := range challenges {
		key := challenge.Name
		lang := challenge.SolutionLang
		if lang == "" {
			lang = "unsolved"
		}
		challengeMap[key] = append(challengeMap[key], lang)
	}

	// Create a sorted list of challenge names
	var sortedChallenges []string
	for challenge := range challengeMap {
		sortedChallenges = append(sortedChallenges, challenge)
	}
	sort.Strings(sortedChallenges)

	// Print sorted challenges with their languages
	for _, challenge := range sortedChallenges {
		languages := challengeMap[challenge]
		sort.Strings(languages) // Sort languages for consistent output
		for _, lang := range languages {
			fmt.Printf("%s %s\n", challenge, lang)
		}
	}

	return nil
}

func setupDataset() error {
	fmt.Println("Downloading dataset...")
	if err := downloadFile(filepath.Join(getCacheDir(), datasetParquet), datasetURL); err != nil {
		return fmt.Errorf("error downloading dataset: %v", err)
	}

	fmt.Println("Processing dataset...")
	challenges, err := processParquetFile(filepath.Join(getCacheDir(), datasetParquet))
	if err != nil {
		return fmt.Errorf("error processing dataset: %v", err)
	}

	fmt.Println("Saving challenges...")
	if err := saveChallenges(challengesFile, challenges); err != nil {
		return fmt.Errorf("error saving challenges: %v", err)
	}

	fmt.Println("Setup complete!")
	return nil
}

func downloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func processParquetFile(filepath string) ([]Challenge, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer f.Close()

	reader, err := file.NewParquetReader(f)
	if err != nil {
		return nil, fmt.Errorf("error creating parquet reader: %v", err)
	}
	defer reader.Close()

	arrowReader, err := pqarrow.NewFileReader(reader, pqarrow.ArrowReadProperties{}, memory.DefaultAllocator)
	if err != nil {
		return nil, fmt.Errorf("error creating arrow reader: %v", err)
	}

	table, err := arrowReader.ReadTable(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error reading table: %v", err)
	}
	defer table.Release()

	numRows := int(table.NumRows())
	fmt.Printf("Total rows in parquet file: %d\n", numRows)

	challenges := make([]Challenge, 0, numRows)

	for i := 0; i < int(table.NumCols()); i++ {
		col := table.Column(i)
		chunks := col.Data().Chunks()

		switch col.DataType().ID() {
		case arrow.STRING:
			for _, chunk := range chunks {
				strArr := array.NewStringData(chunk.Data())
				for j := 0; j < strArr.Len(); j++ {
					if len(challenges) <= j {
						challenges = append(challenges, Challenge{})
					}
					switch i {
					case 0:
						challenges[j].Name = strArr.Value(j)
					case 1:
						challenges[j].Solution = strArr.Value(j)
					case 2:
						challenges[j].Input = strArr.Value(j)
					case 3:
						challenges[j].Task = strArr.Value(j)
					case 4:
						challenges[j].SolutionLang = strArr.Value(j)
					case 6:
						challenges[j].Answer = strArr.Value(j)
					}
				}
			}
		case arrow.INT64:
			for _, chunk := range chunks {
				int64Arr := array.NewInt64Data(chunk.Data())
				for j := 0; j < int64Arr.Len(); j++ {
					if len(challenges) <= j {
						challenges = append(challenges, Challenge{})
					}
					challenges[j].Year = int64Arr.Value(j)
				}
			}
		}

		if i%100 == 0 {
			fmt.Printf("Processed %d columns\n", i)
		}
	}

	fmt.Printf("Total challenges processed: %d\n", len(challenges))
	return challenges, nil
}
