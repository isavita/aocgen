# AoCGen: Advent of Code Solution Generator

AoCGen is a CLI tool designed to streamline the process of downloading Advent of Code tasks, generating solutions in various programming languages, and validating the solutions.

## Installation

To install AoCGen, ensure you have Go installed on your system, then run:

```
go get github.com/isavita/aocgen
```

## Usage

AoCGen supports the following commands:

### Setup

Initialize the dataset:

```
aocgen setup
```

This command downloads and processes the Advent of Code dataset, preparing it for use with other commands.

### List Challenges

View all available challenges:

```
aocgen list
```

### Download Challenge

Download a specific Advent of Code challenge:

```
aocgen download --day <day> --year <year> --session <session_token>
```

- `--day`: The day of the challenge (1-25)
- `--year`: The year of the challenge
- `--session`: Your Advent of Code session token

The challenge will be downloaded and saved in the `aocgen_cache` directory. The challenge description and input will be stored in a JSON file named `challenges.json` within this directory.

Note: The `--part` option is missing from the command. If you want to download a specific part of the challenge (1 or 2), you may need to add this option to the command and update the code accordingly.

### Generate Solution

Generate a solution template for a specific challenge:

```
aocgen generate --day <day> --part <part> --year <year> --lang <language> --model <ai_model> --model_api <api_endpoint>
```

- `--day`: The day of the challenge (1-25)
- `--part`: The part of the challenge (1 or 2)
- `--year`: The year of the challenge
- `--lang`: The programming language for the solution
- `--model`: The AI model to use for generation (e.g., "ollama/codellama")
- `--model_api`: The API endpoint for the AI model

### Evaluate Solution

Evaluate a generated solution:

```
aocgen eval --day <day> --part <part> --year <year> --lang <language>
```

- `--day`: The day of the challenge (1-25)
- `--part`: The part of the challenge (1 or 2)
- `--year`: The year of the challenge
- `--lang`: The programming language of the solution

## Feature Checklist

- [x] Setup dataset
- [x] List available challenges
- [x] Download specific challenges
- [x] Generate solution templates
- [x] Evaluate solutions
- [ ] Support for multiple AI models
- [ ] Automatic submission of solutions to Advent of Code
- [ ] Progress tracking for completed challenges
- [ ] Integration with version control systems
- [ ] Support for custom solution templates
- [ ] Performance benchmarking of solutions

## Contributing

Contributions to AoCGen are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
