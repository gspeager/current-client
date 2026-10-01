package git

import (
	"regexp"
	"strings"
)

type Trailer struct {
	Key   string
	Value string
}

var trailerLine = regexp.MustCompile(`^([A-Za-z0-9][A-Za-z0-9-]*):\s*(.*)$`)

// ParseTrailers reads the trailer block: the body's last paragraph, when every
// line in it is a "Key: value" trailer or an indented continuation of one.
func ParseTrailers(body string) []Trailer {
	paragraphs := strings.Split(strings.TrimSpace(strings.ReplaceAll(body, "\r\n", "\n")), "\n\n")
	last := paragraphs[len(paragraphs)-1]
	if last == "" {
		return nil
	}
	var trailers []Trailer
	for _, line := range strings.Split(last, "\n") {
		if m := trailerLine.FindStringSubmatch(line); m != nil {
			trailers = append(trailers, Trailer{Key: m[1], Value: strings.TrimSpace(m[2])})
			continue
		}
		if len(trailers) == 0 || strings.TrimLeft(line, " \t") == line {
			return nil
		}
		trailers[len(trailers)-1].Value += " " + strings.TrimSpace(line)
	}
	return trailers
}

func CoAuthors(trailers []Trailer) []Author {
	var authors []Author
	for _, t := range trailers {
		if !strings.EqualFold(t.Key, "Co-authored-by") {
			continue
		}
		name, email := t.Value, ""
		if open := strings.LastIndex(t.Value, "<"); open >= 0 && strings.HasSuffix(t.Value, ">") {
			name, email = strings.TrimSpace(t.Value[:open]), t.Value[open+1:len(t.Value)-1]
		}
		authors = append(authors, Author{Name: name, Email: email})
	}
	return authors
}
