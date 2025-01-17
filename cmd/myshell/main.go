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

	// Handle backslashes in file paths
	for i, arg := range args {
		args[i] = strings.ReplaceAll(arg, "\\", "")
	}

	return args
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
		command := parts[0]

		switch command {
		case "exit":
			if len(parts) > 1 {
				exitCode, err := strconv.Atoi(parts[1])
				if err != nil {
					fmt.Fprintln(os.Stderr, "Invalid exit code:", parts[1])
					continue
				}
				os.Exit(exitCode)
			}
			os.Exit(0)
		case "echo":
			echo(parts[1:])
		case "type":
			typeCmd(parts[1:])
		case "pwd":
			pwd()
		case "cd":
			cd(parts[1:])
		default:
			// Search for the command in PATH
			pathDirs := strings.Split(os.Getenv("PATH"), ":")
			found := false

			for _, dir := range pathDirs {
				filePath := filepath.Join(dir, command)
				if info, err := os.Stat(filePath); err == nil && !info.IsDir() {
					if info.Mode()&0111 != 0 {
						// Execute the command using the original command name
						cmd := exec.Command(filePath)
						cmd.Args = append([]string{command}, parts[1:]...)
						cmd.Stdout = os.Stdout
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
	}
}
