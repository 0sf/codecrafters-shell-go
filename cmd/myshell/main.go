package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func echo(args []string) {
	fmt.Println(strings.Join(args, " "))
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
		parts := strings.Split(input, " ")
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
		default:
			fmt.Fprintf(os.Stdout, "%s: command not found\n", input)
		}
	}
}
