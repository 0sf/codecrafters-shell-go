package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/term"
)

func echo(args []string) {
	fmt.Println(strings.Join(args, " "))
}

func pwd() {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting current directory:", err)
		return
	}
	fmt.Println(dir)
}

func cd(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: cd directory")
		return
	}

	path := args[0]
	if path == "~" {
		// Get home directory from HOME environment variable
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			fmt.Fprintln(os.Stderr, "cd: HOME environment variable not set")
			return
		}
		path = homeDir
	}

	err := os.Chdir(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cd: %s: No such file or directory\n", args[0])
	}
}

func typeCmd(args []string) {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: type command_name")
		return
	}

	switch args[0] {
	case "exit", "echo", "type", "pwd", "cd":
		fmt.Printf("%s is a shell builtin\n", args[0])
		return
	}

	// Search in PATH
	pathDirs := strings.Split(os.Getenv("PATH"), ":")
	for _, dir := range pathDirs {
		filePath := filepath.Join(dir, args[0])
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			// Check if the file is executable
			if info.Mode()&0111 != 0 {
				fmt.Printf("%s is %s\n", args[0], filePath)
				return
			}
		}
	}

	fmt.Printf("%s: not found\n", args[0])
}

func parseInput(input string) []string {
	var args []string
	var currentArg strings.Builder
	inSingleQuotes, inDoubleQuotes := false, false
	escaped := false

	for i := 0; i < len(input); i++ {
		char := input[i]

		if escaped {
			// If previous char was backslash, add current char literally
			currentArg.WriteByte(char)
			escaped = false
			continue
		}

		switch char {
		case '\\':
			if inSingleQuotes {
				// Backslashes are treated literally in single quotes
				currentArg.WriteByte(char)
			} else if inDoubleQuotes {
				// In double quotes, only escape \ and "
				if i+1 < len(input) && (input[i+1] == '\\' || input[i+1] == '"') {
					escaped = true
				} else {
					currentArg.WriteByte(char)
				}
			} else {
				escaped = true
			}
		case '"':
			if !inSingleQuotes {
				inDoubleQuotes = !inDoubleQuotes
			} else {
				currentArg.WriteByte(char)
			}
		case '\'':
			if !inDoubleQuotes {
				inSingleQuotes = !inSingleQuotes
			} else {
				currentArg.WriteByte(char)
			}
		case ' ':
			if inSingleQuotes || inDoubleQuotes {
				currentArg.WriteByte(char)
			} else if escaped {
				currentArg.WriteByte(char)
				escaped = false
			} else {
				if currentArg.Len() > 0 {
					args = append(args, currentArg.String())
					currentArg.Reset()
				}
			}
		default:
			currentArg.WriteByte(char)
		}
	}

	// Add the last argument if exists
	if currentArg.Len() > 0 {
		args = append(args, currentArg.String())
	}

	return args
}

func executeCommand(command string, args []string, outputFile, errorFile string, appendOutput, appendError bool) {
	pathDirs := strings.Split(os.Getenv("PATH"), ":")
	found := false

	for _, dir := range pathDirs {
		filePath := filepath.Join(dir, command)
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			if info.Mode()&0111 != 0 {
				// Set up stdout redirection
				var stdout *os.File = os.Stdout
				if outputFile != "" {
					flag := os.O_WRONLY | os.O_CREATE
					if appendOutput {
						flag |= os.O_APPEND
					} else {
						flag |= os.O_TRUNC
					}
					var err error
					stdout, err = os.OpenFile(outputFile, flag, 0644)
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error opening output file:", err)
						return
					}
					defer stdout.Close()
				}

				// Set up stderr redirection
				var stderr *os.File = os.Stderr
				if errorFile != "" {
					flag := os.O_WRONLY | os.O_CREATE
					if appendError {
						flag |= os.O_APPEND
					} else {
						flag |= os.O_TRUNC
					}
					var err error
					stderr, err = os.OpenFile(errorFile, flag, 0644)
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error creating error file:", err)
						return
					}
					defer stderr.Close()
				}

				cmd := exec.Command(filePath)
				cmd.Args = append([]string{command}, args...)
				cmd.Stdout = stdout
				cmd.Stderr = stderr

				cmd.Run()
				found = true
				break
			}
		}
	}

	if !found {
		fmt.Fprintf(os.Stdout, "%s: command not found\n", command)
	}
}

func main() {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	terminal := term.NewTerminal(os.Stdin, "$ ")

	// Set up autocomplete callback
	terminal.AutoCompleteCallback = func(line string, pos int, key rune) (newLine string, newPos int, ok bool) {
		if key != '\t' {
			return "", 0, false
		}

		// Get the word being completed (from last space to cursor)
		word := line[:pos]
		if lastSpace := strings.LastIndex(word, " "); lastSpace >= 0 {
			word = word[lastSpace+1:]
		}

		// Try to find matching commands
		var matches []string
		for _, cmd := range []string{"exit", "echo", "type", "pwd", "cd"} {
			if strings.HasPrefix(cmd, word) {
				matches = append(matches, cmd)
			}
		}

		// If exactly one match, complete it
		if len(matches) == 1 {
			suffix := matches[0][len(word):]
			newLine = line[:pos] + suffix + " " + line[pos:]
			newPos = pos + len(suffix) + 1
			return newLine, newPos, true
		}

		return "", 0, false
	}

	for {
		line, err := terminal.ReadLine()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if len(line) == 0 {
			continue
		}

		// Rest of your existing command processing code
		parts := parseInput(line)
		if len(parts) == 0 {
			continue
		}

		// Find redirection operators
		outputFile := ""
		errorFile := ""
		appendOutput := false
		appendError := false
		cmdParts := parts

		for i := 0; i < len(parts); i++ {
			if (parts[i] == ">" || parts[i] == "1>") && i+1 < len(parts) {
				outputFile = parts[i+1]
				appendOutput = false
				if &cmdParts[0] == &parts[0] {
					cmdParts = make([]string, i)
					copy(cmdParts, parts[:i])
				}
				break
			} else if (parts[i] == ">>" || parts[i] == "1>>") && i+1 < len(parts) {
				outputFile = parts[i+1]
				appendOutput = true
				if &cmdParts[0] == &parts[0] {
					cmdParts = make([]string, i)
					copy(cmdParts, parts[:i])
				}
				break
			} else if parts[i] == "2>" && i+1 < len(parts) {
				errorFile = parts[i+1]
				appendError = false
				if &cmdParts[0] == &parts[0] {
					cmdParts = make([]string, i)
					copy(cmdParts, parts[:i])
				}
				break
			} else if parts[i] == "2>>" && i+1 < len(parts) {
				errorFile = parts[i+1]
				appendError = true
				if &cmdParts[0] == &parts[0] {
					cmdParts = make([]string, i)
					copy(cmdParts, parts[:i])
				}
				break
			}
		}

		command := cmdParts[0]
		args := cmdParts[1:]

		switch command {
		case "exit":
			if len(args) > 0 {
				exitCode, err := strconv.Atoi(args[0])
				if err != nil {
					fmt.Fprintln(os.Stderr, "Invalid exit code:", args[0])
					continue
				}
				os.Exit(exitCode)
			}
			os.Exit(0)
		case "echo":
			if outputFile != "" || errorFile != "" {
				stdout := os.Stdout
				stderr := os.Stderr

				if outputFile != "" {
					flag := os.O_WRONLY | os.O_CREATE
					if appendOutput {
						flag |= os.O_APPEND
					} else {
						flag |= os.O_TRUNC
					}
					if f, err := os.OpenFile(outputFile, flag, 0644); err == nil {
						stdout = f
						defer f.Close()
					} else {
						fmt.Fprintln(stderr, "Error opening output file:", err)
						continue
					}
				}
				if errorFile != "" {
					flag := os.O_WRONLY | os.O_CREATE
					if appendError {
						flag |= os.O_APPEND
					} else {
						flag |= os.O_TRUNC
					}
					if f, err := os.OpenFile(errorFile, flag, 0644); err == nil {
						stderr = f
						defer f.Close()
					} else {
						fmt.Fprintln(os.Stderr, "Error creating error file:", err)
						continue
					}
				}
				fmt.Fprintln(stdout, strings.Join(args, " "))
			} else {
				echo(args)
			}
		case "type", "pwd":
			stdout := os.Stdout
			stderr := os.Stderr
			if outputFile != "" {
				flag := os.O_WRONLY | os.O_CREATE
				if appendOutput {
					flag |= os.O_APPEND
				} else {
					flag |= os.O_TRUNC
				}
				if f, err := os.OpenFile(outputFile, flag, 0644); err == nil {
					stdout = f
					defer f.Close()
				} else {
					fmt.Fprintln(os.Stderr, "Error opening output file:", err)
					continue
				}
			}
			if errorFile != "" {
				flag := os.O_WRONLY | os.O_CREATE
				if appendError {
					flag |= os.O_APPEND
				} else {
					flag |= os.O_TRUNC
				}
				if f, err := os.OpenFile(errorFile, flag, 0644); err == nil {
					stderr = f
					defer f.Close()
				} else {
					fmt.Fprintln(os.Stderr, "Error creating error file:", err)
					continue
				}
			}

			oldStdout := os.Stdout
			oldStderr := os.Stderr
			os.Stdout = stdout
			os.Stderr = stderr

			if command == "type" {
				typeCmd(args)
			} else {
				pwd()
			}

			os.Stdout = oldStdout
			os.Stderr = oldStderr
		case "cd":
			cd(args)
		default:
			executeCommand(command, args, outputFile, errorFile, appendOutput, appendError)
		}
	}
}
