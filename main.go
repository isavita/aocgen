package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
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

var cacheDir = "aocgen_cache"

const challengesFile = "challenges.json"
const datasetParquet = "dataset.parquet"
const datasetURL = "https://huggingface.co/datasets/isavita/advent-of-code/resolve/refs%2Fconvert%2Fparquet/default/train/0000.parquet"

func init() {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		fmt.Printf("Failed to create cache directory: %v\n", err)
		os.Exit(1)
	}
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
		fmt.Println("Expected 'generate', 'download', 'eval', 'list', or 'setup' subcommands")
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
		day := evalCmd.Int("day", 0, "Day of the challenge")
		part := evalCmd.Int("part", 0, "Part of the challenge")
		year := evalCmd.Int("year", 0, "Year of the challenge")
		lang := evalCmd.String("lang", "", "Programming language of the solution")

		evalCmd.Parse(os.Args[2:])

		flags := Flags{
			Day:  *day,
			Part: *part,
			Year: *year,
			Lang: *lang,
		}

		if err := runEvaluationCommand(flags); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "list":
		if err := listChallenges(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "setup":
		if err := setupDataset(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Println("Expected 'generate', 'download' or 'eval' subcommands")
		os.Exit(1)
	}
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

	// Save the challenge to the JSON file
	challenges, err := loadChallenges(cacheDir, "challenges.json")
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

	return os.WriteFile(filepath.Join(cacheDir, filename), data, 0644)
}

func runGenerateCommand(flags Flags) error {
	challenges, err := loadChallenges(cacheDir, "challenges.json")
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
	err := downloadChallenge(flags)
	if err != nil {
		return fmt.Errorf("error downloading challenge: %v", err)
	}

	return nil
}

func evaluateSolution(challenge Challenge, solutionPath string, lang string, timeout time.Duration) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var cmd *exec.Cmd
	switch lang {
	case "python":
		cmd = exec.CommandContext(ctx, "python", solutionPath)
	case "javascript":
		cmd = exec.CommandContext(ctx, "node", solutionPath)
	case "ruby":
		cmd = exec.CommandContext(ctx, "ruby", solutionPath)
	case "go":
		cmd = exec.CommandContext(ctx, "go", "run", solutionPath)
	case "java":
		cmd = exec.CommandContext(ctx, "java", solutionPath)
	case "elixir":
		cmd = exec.CommandContext(ctx, "elixir", solutionPath)
	// Add more cases for other languages as needed
	default:
		return false, fmt.Errorf("unsupported language for execution: %s", lang)
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return false, fmt.Errorf("execution timed out after %v", timeout)
		}
		return false, fmt.Errorf("execution failed: %v\nStderr: %s", err, errBuf.String())
	}

	output := outBuf.String()
	return validSolution(output, challenge.Answer), nil
}

func runEvaluationCommand(flags Flags) error {
	challenges, err := loadChallenges(cacheDir, "challenges.json")
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

	correct, err := evaluateSolution(challenge, solutionPath, flags.Lang, 20*time.Second)
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

func validSolution(result, answer string) bool {
	if strings.Contains(result, answer) {
		return true
	}

	// Check for ASCII art answers
	asciiPatterns := []string{
		".##..####.###..#..#.###..####.###....##.###...###.",
		" ##  #### ###  #  # ###  #### ###    ## ###   ### ",
		"#....#..#....#.....###..######....##....#....#....##....######",
		"#    #  #    #     ###  ######    ##    #    #    ##    ######",
		"####.###..####.#..#.###..\n#....#..#....#.#..#.#..#.",
		"#### ###  #### #  # ###  \n#    #  #    # #  # #  # ",
		"..#....###....##.#..#.####.#..#.#....#..#.\n",
		" #    ###    ## #  # #### #  # #    #  # \n",
		" █    ███    ██ █  █ ████ █  █ █    █  █ \n",
		"#..#.#..#.#..#.#..#.#..#.#..#.#..#....#",
		"#  # #  # #  # #  # #  # #  # #  #    #",
		"###..###..###...##..###...##...##..####.",
		"###  ###  ###   ##  ###   ##   ##  #### ",
	}

	for _, pattern := range asciiPatterns {
		if strings.Contains(result, pattern) {
			return true
		}
	}

	// Check for specific numeric formats
	if strings.Contains(result, "3.465154e+06") || strings.Contains(result, "3.465154e+6") {
		return true
	}

	return false
}

func listChallenges() error {
	challenges, err := loadChallenges(cacheDir, "challenges.json")
	if err != nil {
		return fmt.Errorf("error loading challenges: %v", err)
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
	if err := downloadFile(filepath.Join(cacheDir, datasetParquet), datasetURL); err != nil {
		return fmt.Errorf("error downloading dataset: %v", err)
	}

	fmt.Println("Processing dataset...")
	challenges, err := processParquetFile(filepath.Join(cacheDir, datasetParquet))
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
