# AoCGen: Advent of Code Solution Generator

AoCGen is a CLI tool designed to streamline the process of downloading Advent of Code tasks, generating solutions in various programming languages, and validating the solutions.

## Installation

To install AoCGen, ensure you have Go installed on your system, then run:

```bash
go get github.com/isavita/aocgen
```

## Usage

AoCGen supports the following commands:

### Setup

Initialize the dataset:

```bash
aocgen setup
```

This command downloads and processes the Advent of Code dataset, preparing it for use with other commands.

### List Challenges

View all available challenges:

```bash
aocgen list
```

### Download Challenge

Download a specific Advent of Code challenge:

```bash
aocgen download --day <day> --year <year> --session <session_token>
```

- `--day`: The day of the challenge (1-25)
- `--year`: The year of the challenge
- `--session`: Your Advent of Code session token

### Generate Solution

Generate a solution template for a specific challenge:

```bash
aocgen generate --day <day> --part <part> --year <year> --lang <language> --model <ai_model> --model_api <api_endpoint>
```

- `--day`: The day of the challenge (1-25)
- `--part`: The part of the challenge (1 or 2)
- `--year`: The year of the challenge
- `--lang`: The programming language for the solution
- `--model`: The AI model to use for generation
- `--model_api`: The API endpoint for the AI model

#### Supported AI Models

AoCGen supports multiple AI models for solution generation. Here are examples for each supported model:

1. OpenAI GPT Models:
```bash
aocgen generate --day 1 --part 1 --year 2023 --lang python --model gpt-4o-mini --model_api https://api.openai.com/v1/chat/completions
```

2. Ollama Models:
```bash
aocgen generate --day 1 --part 1 --year 2015 --lang python --model ollama/mistral-nemo --model_api http://localhost:11434/v1/chat/completions
```

3. Groq Models:
```bash
aocgen generate --day 1 --part 1 --year 2023 --lang python --model groq/mixtral-8x7b-32768 --model_api https://api.groq.com/openai/v1/chat/completions
```

### Evaluate Solution

Evaluate a generated solution:

```bash
aocgen eval --day <day> --part <part> --year <year> --lang <language>
```

- `--day`: The day of the challenge (1-25)
- `--part`: The part of the challenge (1 or 2)
- `--year`: The year of the challenge
- `--lang`: The programming language of the solution

### Performance Benchmark

Run performance benchmarks for solutions in a specific language:

```bash
aocgen perf --lang <language> --timeout <timeout_milliseconds>
```

## Feature Checklist

- [x] Setup dataset
- [x] List available challenges
- [x] Download specific challenges
- [x] Generate solution templates
- [x] Evaluate solutions
- [x] Support for multiple AI models
- [ ] Automatic submission of solutions to Advent of Code
- [ ] Progress tracking for completed challenges
- [ ] Integration with version control systems
- [ ] Support for custom solution templates
- [x] Performance benchmarking of solutions

## Running Tests

AoCGen uses environment variables to control certain aspects of testing. You can create a `.env` file in the project root to set these variables. Here's an example:

```bash
ADVENT_OF_CODE_SESSION=your_session_token_here
SKIP_OPENAI_TESTS=1
SKIP_OLLAMA_TESTS=1
SKIP_GROQ_TESTS=1
```

- `ADVENT_OF_CODE_SESSION`: Your Advent of Code session token for downloading challenges.
- `SKIP_OPENAI_TESTS`: Set to 1 to skip OpenAI API tests.
- `SKIP_OLLAMA_TESTS`: Set to 1 to skip Ollama API tests.
- `SKIP_GROQ_TESTS`: Set to 1 to skip Groq API tests.

To run the tests with these settings:

```bash
go test ./...
```

Note: Make sure to keep your `.env` file private and not commit it to version control.

## Contributing

Contributions to AoCGen are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
