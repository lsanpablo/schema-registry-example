package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	const baseDir = "events"
	const outputFile = "index.md"

	// topLevel -> subject -> []versionLinks
	grouped := make(map[string]map[string][]string)

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".md") || !strings.HasPrefix(filepath.Base(path), "v") {
			return nil
		}

		relPath := filepath.ToSlash(path)
		subjectPath := strings.TrimPrefix(filepath.Dir(relPath), baseDir+"/")
		subjectParts := strings.Split(subjectPath, "/")
		if len(subjectParts) < 2 {
			return nil // skip unexpected paths like ./events/foo/v1.md
		}

		topLevel := subjectParts[0]
		subject := strings.Join(subjectParts, ".")

		version := strings.TrimSuffix(filepath.Base(relPath), ".md")
		link := fmt.Sprintf("[%s](./%s)", version, relPath)

		if _, ok := grouped[topLevel]; !ok {
			grouped[topLevel] = make(map[string][]string)
		}
		grouped[topLevel][subject] = append(grouped[topLevel][subject], link)

		return nil
	})

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error walking events directory: %v\n", err)
		os.Exit(1)
	}

	// Build markdown
	var b strings.Builder
	b.WriteString("# Event Index\n\n")

	topLevels := make([]string, 0, len(grouped))
	for top := range grouped {
		topLevels = append(topLevels, top)
	}
	sort.Strings(topLevels)

	for _, top := range topLevels {
		b.WriteString(fmt.Sprintf("## %s\n\n", top))
		b.WriteString("| Subject | Versions |\n")
		b.WriteString("|---------|----------|\n")

		subjects := grouped[top]
		subjectKeys := make([]string, 0, len(subjects))
		for subj := range subjects {
			subjectKeys = append(subjectKeys, subj)
		}
		sort.Strings(subjectKeys)

		for _, subj := range subjectKeys {
			links := subjects[subj]
			sort.Strings(links)
			b.WriteString(fmt.Sprintf("| `%s` | %s |\n", subj, strings.Join(links, ", ")))
		}
		b.WriteString("\n")
	}

	err = os.WriteFile(outputFile, []byte(b.String()), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing index.md: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Generated index.md")
}
