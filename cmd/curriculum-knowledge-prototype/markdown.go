package main

import "strings"

type markdownSection struct {
	title string
	text  string
}

func splitMarkdownSections(markdown string) []markdownSection {
	lines := strings.Split(markdown, "\n")
	sections := []markdownSection{}
	title := "Overview"
	body := []string{}

	flush := func() {
		text := strings.TrimSpace(strings.Join(body, "\n"))
		if text == "" {
			return
		}
		sections = append(sections, markdownSection{
			title: title,
			text:  text,
		})
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			flush()
			title = strings.TrimSpace(strings.TrimLeft(line, "#"))
			body = body[:0]
			continue
		}
		body = append(body, line)
	}
	flush()

	if len(sections) == 0 && strings.TrimSpace(markdown) != "" {
		return []markdownSection{{
			title: "Notes",
			text:  markdown,
		}}
	}
	return sections
}
