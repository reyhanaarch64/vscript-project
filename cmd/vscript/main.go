package main

import (
	"fmt"
	"os"

	"vscript/engine"
)

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		printUsage()
		os.Exit(2)
	}
	if args[0] == "--version" || args[0] == "-v" {
		fmt.Println("vscript 0.2.0")
		return
	}
	if args[0] == "paths" {
		fmt.Println(engine.FFmpegSearchPaths()[0])
		return
	}

	file := args[0]
	if file == "run" {
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "kesalahan: file sumber belum diberikan")
			printUsage()
			os.Exit(2)
		}
		file = args[1]
	}

	source, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kesalahan membaca %s: %v\n", file, err)
		os.Exit(1)
	}

	if err := engine.Run(string(source), file); err != nil {
		fmt.Fprintf(os.Stderr, "kesalahan: %v\n", err)
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("penggunaan: vscript <file.vs>")
	fmt.Println("           vscript run <file.vs>")
	fmt.Println("           vscript paths")
	fmt.Println("           vscript --version")
}
