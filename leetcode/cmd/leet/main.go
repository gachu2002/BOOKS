package main

import (
	"fmt"
	"leetcode/internal/registry"
	"os"
	"sort"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		listQuestions()
	case "show":
		requireSlugAndRun(showQuestion)
	case "run":
		requireSlugAndRun(runQuestion)
	default:
		printUsage()
		os.Exit(1)
	}
}

func requireSlugAndRun(fn func(string) error) {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "missing question slug")
		printUsage()
		os.Exit(1)
	}

	if err := fn(strings.TrimSpace(os.Args[2])); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func listQuestions() {
	questions := registry.All()
	sort.Slice(questions, func(i, j int) bool {
		if questions[i].Category == questions[j].Category {
			return questions[i].Slug < questions[j].Slug
		}
		return questions[i].Category < questions[j].Category
	})

	for _, q := range questions {
		fmt.Printf("%-20s %-18s %s\n", q.Category, q.Slug, q.Title)
	}
	if len(questions) == 0 {
		fmt.Println("no registered questions")
	}
}

func showQuestion(slug string) error {
	q, ok := registry.Get(slug)
	if !ok {
		return fmt.Errorf("unknown question: %s", slug)
	}

	fmt.Printf("Title:    %s\n", q.Title)
	fmt.Printf("Slug:     %s\n", q.Slug)
	fmt.Printf("Category: %s\n", q.Category)
	fmt.Printf("URL:      %s\n", q.URL)
	return nil
}

func runQuestion(slug string) error {
	q, ok := registry.Get(slug)
	if !ok {
		return fmt.Errorf("unknown question: %s", slug)
	}

	fmt.Printf("Running %s (%s)\n", q.Title, q.Slug)
	fmt.Printf("Category: %s\n", q.Category)
	fmt.Printf("URL:      %s\n\n", q.URL)

	if err := q.Run(); err != nil {
		return fmt.Errorf("run %s: %w", slug, err)
	}

	return nil
}

func printUsage() {
	fmt.Println("usage:")
	fmt.Println("  go run ./cmd/leet list")
	fmt.Println("  go run ./cmd/leet show <slug>")
	fmt.Println("  go run ./cmd/leet run <slug>")
}
