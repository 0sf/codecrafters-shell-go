package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func main() {
	reader := bufio.NewReader(os.Stdin)

	for {
		// Print the prompt
		fmt.Fprint(os.Stdout, "$ ")

		// Read user input
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error reading input:", err)
			continue
		}

		// Trim the newline character
		input = strings.TrimSpace(input)

		// Split the input into command and arguments
		parts := strings.Split(input, " ")
		command := parts[0]

		// Handle the exit command
		if command == "exit" {
			if len(parts) > 1 {
				exitCode, err := strconv.Atoi(parts[1])
				if err != nil {
					fmt.Fprintln(os.Stderr, "Invalid exit code:", parts[1])
					continue
				}
				os.Exit(exitCode)
			} else {
				os.Exit(0)
			}
		}

		// Print the command not found message
		fmt.Fprint(os.Stdout, input+": command not found\n")
	}
}
