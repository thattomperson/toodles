package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"strings"

	"github.com/rjeczalik/notify"
)

type Todo struct {
	File string
	Line int
}

func main() {

	todos := make(chan Todo, 10)

	go func() {
		for l := range todos {
			fmt.Printf("%s:%d\n", l.File, l.Line)
		}
	}()

	c := make(chan notify.EventInfo, 10)

	for i := 0; i < runtime.NumCPU()*2; i++ {
		go func() {
			for e := range c {
				wd, err := os.Getwd()
				if err != nil {
					checkFile(e.Path(), todos)
					continue
				}

				p, err := filepath.Rel(wd, e.Path())
				if err != nil {
					p = e.Path()
				}
				checkFile(p, todos)
			}
		}()
	}

	if err := notify.Watch("./...", c, notify.All); err != nil {
		log.Fatal(err)
	}
	defer notify.Stop(c)

	firstScan(todos)

	for t := range todos {
		fmt.Printf("%s:%d\n", t.File, t.Line)
	}

	close(c)
	close(todos)
}

func checkFile(path string, c chan Todo) {
	f, err := os.Open(path)
	if err != nil {
		log.Printf("%s: %v\n", path, err)
	}

	// Splits on newlines by default.
	scanner := bufio.NewScanner(f)

	line := 1
	// https://golang.org/pkg/bufio/#Scanner.Scan
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), os.Args[1]) {
			c <- Todo{
				File: path,
				Line: line,
			}
		}

		line++
	}

	if err := scanner.Err(); err != nil {
		log.Printf("%s: %v\n", path, err)
	}
}

func ignoredFile(name string) bool {
	switch filepath.Ext(name) {
	case ".map":
		return true
	}

	return false
}

func ignoredDir(name string) bool {
	switch name {
	case "node_modules":
		return true
	case "public":
		return true
	case "vendor":
		return true
	case "debugbar":
		return true
	case "themes":
		return true
	case ".git":
		return true
	}

	return false
}

func firstScan(todos chan Todo) {
	filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("%s: %v", path, err)
		}

		if info.IsDir() && ignoredDir(info.Name()) {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			if ignoredFile(info.Name()) {
				return nil
			}

			checkFile(path, todos)
		}

		return nil
	})

}
