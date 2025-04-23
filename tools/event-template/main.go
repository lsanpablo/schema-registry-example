package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed template/event.md.tmpl
var tmplFS embed.FS

type EventTemplateData struct {
	Subject string
	Version string
}

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <subject> <version>\n", os.Args[0])
		os.Exit(1)
	}

	subject := os.Args[1]
	version := os.Args[2]

	parts := strings.Split(subject, ".")
	if len(parts) < 2 {
		fmt.Fprintln(os.Stderr, "Subject must be like item.added or foo.bar")
		os.Exit(1)
	}

	dirPath := filepath.Join(append([]string{"events"}, parts...)...)
	mdPath := filepath.Join(dirPath, "v"+version+".md")

	if err := os.MkdirAll(dirPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create directory: %v\n", err)
		os.Exit(1)
	}

	if _, err := os.Stat(mdPath); err == nil {
		fmt.Fprintf(os.Stderr, "File already exists: %s\n", mdPath)
		os.Exit(1)
	}

	templateContent, err := tmplFS.ReadFile("template/event.md.tmpl")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load template: %v\n", err)
		os.Exit(1)
	}

	tmpl := template.Must(template.New("event").Parse(string(templateContent)))

	f, err := os.Create(mdPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write markdown file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	err = tmpl.Execute(f, EventTemplateData{Subject: subject, Version: version})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to execute template: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Created %s\n", mdPath)
}
