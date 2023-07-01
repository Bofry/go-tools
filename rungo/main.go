package main

import (
	"fmt"
	"os"
	"os/exec"
)

var (
	osExit func(int) = os.Exit
)

func main() {
	var (
		pos  int = 0
		args     = []string{"godotenv"}
	)

	// parse godotenv flag
	flag := shift(&pos)
	switch flag {
	case "-f":
		args = append(args, flag, shift(&pos))
		args = append(args, "go", "run")
		args = append(args, arguments(&pos)...)
	case ".":
		// skip
	default:
		args = append(args, "go", "run", flag)
		args = append(args, arguments(&pos)...)
	}

	err := executeCommand(args[0], args[1:]...)
	if err != nil {
		throw(err.Error())
		exit(1)
	}
}

func throw(err string) {
	fmt.Fprintln(os.Stderr, err)
}

func exit(code int) {
	osExit(code)
}

func executeCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shift(pos *int) string {
	if *pos < len(os.Args)-1 {
		*pos++
		return os.Args[*pos]
	}
	return ""
}

func arguments(pos *int) []string {
	if *pos < len(os.Args) {
		return os.Args[*pos+1:]
	}
	return nil
}
