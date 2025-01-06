package main

import (
	"bufio"
	"fmt"
	"os"
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

		// Check for exit command
		if input == "exit" {
			break
		}

		// Print the command not found message
		fmt.Fprint(os.Stdout, input+": command not found\n")
	}
}
