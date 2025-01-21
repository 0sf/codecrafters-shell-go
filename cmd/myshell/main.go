package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
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

func executeCommand(command string, args []string, outputFile string) {
	// Search for the command in PATH
	pathDirs := strings.Split(os.Getenv("PATH"), ":")
	found := false

	for _, dir := range pathDirs {
		filePath := filepath.Join(dir, command)
		if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
			if info.Mode()&0111 != 0 {
				// Create or truncate the output file if specified
				var stdout *os.File
				if outputFile != "" {
					var err error
					stdout, err = os.Create(outputFile)
					if err != nil {
						fmt.Fprintln(os.Stderr, "Error creating output file:", err)
						return
					}
					defer stdout.Close()
				} else {
					stdout = os.Stdout
				}

				// Execute the command using the original command name
				cmd := exec.Command(filePath)
				cmd.Args = append([]string{command}, args...)
				cmd.Stdout = stdout
				cmd.Stderr = os.Stderr

				if err := cmd.Run(); err != nil {
					fmt.Fprintln(os.Stderr, "Error executing command:", err)
				}
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
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Fprint(os.Stdout, "$ ")

		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			continue
		}

		input = strings.TrimSpace(input)
		parts := parseInput(input)
		if len(parts) == 0 {
			continue
		}

		// Find redirection operator
		outputFile := ""
		cmdParts := parts
		for i, part := range parts {
			if part == ">" || part == "1>" {
				if i+1 < len(parts) {
					outputFile = parts[i+1]
					cmdParts = parts[:i]
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
			if outputFile != "" {
				if f, err := os.Create(outputFile); err == nil {
					fmt.Fprintln(f, strings.Join(args, " "))
					f.Close()
				} else {
					fmt.Fprintln(os.Stderr, "Error creating output file:", err)
				}
			} else {
				echo(args)
			}
		case "type":
			if outputFile != "" {
				if f, err := os.Create(outputFile); err == nil {
					old := os.Stdout
					os.Stdout = f
					typeCmd(args)
					os.Stdout = old
					f.Close()
				} else {
					fmt.Fprintln(os.Stderr, "Error creating output file:", err)
				}
			} else {
				typeCmd(args)
			}
		case "pwd":
			if outputFile != "" {
				if f, err := os.Create(outputFile); err == nil {
					old := os.Stdout
					os.Stdout = f
					pwd()
					os.Stdout = old
					f.Close()
				} else {
					fmt.Fprintln(os.Stderr, "Error creating output file:", err)
				}
			} else {
				pwd()
			}
		case "cd":
			cd(args)
		default:
			executeCommand(command, args, outputFile)
		}
	}
}
