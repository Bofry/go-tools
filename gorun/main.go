package main

import (
	"fmt"
	"os"
	"os/exec"
)

var (
	pos int = 0
)

func main() {
	var (
		args = []string{"godotenv"}
	)

	// parse godotenv flag
	flag := shift()
	switch flag {
	case "-f":
		args = append(args, flag, shift())
		args = append(args, "go", "run")
		args = append(args, arguments()...)
	case ".":
		// skip
	default:
		args = append(args, "go", "run", flag)
		args = append(args, arguments()...)
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		throw(err.Error())
		os.Exit(1)
	}
}

func throw(err string) {
	fmt.Fprintln(os.Stderr, err)
}

func shift() string {
	if pos < len(os.Args)-1 {
		pos++
		return os.Args[pos]
	}
	return ""
}

func arguments() []string {
	if pos < len(os.Args) {
		return os.Args[pos+1:]
	}
	return nil
}
