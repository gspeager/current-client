package notes

import "testing"

func TestLineIn(t *testing.T) {
	const content = "func a() {\n\tone()\n}\n\nfunc b() {\n\ttwo()\n}\n"
	on := func(line int) Note {
		note := Note{Kind: KindReview, Line: line}
		note.Anchor([]byte(content))
		return note
	}
	for name, c := range map[string]struct {
		note    Note
		content string
		line    int
	}{
		"unchanged":             {on(6), content, 6},
		"lines added above":     {on(6), "// new\n\n" + content, 8},
		"CRLF":                  {on(6), "func b() {\r\n\ttwo()\r\n}\r\n", 2},
		"its line changed":      {on(6), "func b() {\n\tthree()\n}\n", 0},
		"its line gone":         {on(6), "", 0},
		"a brace":               {on(7), "// new\n" + content, 8},
		"a brace's block gone":  {on(7), "func a() {\n\tone()\n}\n", 0},
		"the nearest of two":    {Note{Line: 5, LineText: "x1"}, "x1\n\n\n\nx1\n", 5},
		"the first line":        {on(1), content, 1},
		"a brace starting it":   {Note{Line: 1, LineText: "{"}, "{\n{\n", 1},
		"a line of the old one": {Note{OldLine: 2, LineText: "\tone()"}, content, 2},
	} {
		line, found := c.note.LineIn([]byte(c.content))
		if line != c.line || found != (c.line > 0) {
			t.Errorf("%s: LineIn = %d, %v, want %d", name, line, found, c.line)
		}
	}
}
