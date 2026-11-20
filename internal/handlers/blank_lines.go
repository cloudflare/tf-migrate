package handlers

import (
	"bytes"
)

// blankLineLimitsKey is the ctx.Metadata key under which ResourceTransformHandler
// stores the original file's blank-line layout when it removes blocks.
const blankLineLimitsKey = "blank_line_limits"

// blankLineLimits records the blank-line layout of the file before any block
// was removed: how many blank lines it started with, and its longest run of
// blank lines anywhere else.
type blankLineLimits struct {
	leading int
	maxRun  int
}

func isBlank(line []byte) bool {
	return len(bytes.TrimSpace(line)) == 0
}

func measureBlankLines(content []byte) blankLineLimits {
	lines := bytes.Split(content, []byte("\n"))
	var l blankLineLimits
	i := 0
	for i < len(lines) && isBlank(lines[i]) {
		i++
	}
	l.leading = i
	run := 0
	for ; i < len(lines); i++ {
		if isBlank(lines[i]) {
			run++
			if run > l.maxRun {
				l.maxRun = run
			}
		} else {
			run = 0
		}
	}
	return l
}

// collapseBlankLines removes the blank lines that removing blocks leaves
// behind. hclwrite's RemoveBlock keeps the newline that separated each removed
// block, so a file whose blocks are all replaced gains one blank line per
// block at the spot where they used to be (134 for a file of 133 records).
//
// Only runs longer than anything in the original file are shortened, so blank
// lines the user wrote (including inside heredocs) are left alone.
func collapseBlankLines(content []byte, l blankLineLimits) []byte {
	maxRun := l.maxRun
	if maxRun < 1 {
		maxRun = 1
	}
	trailingNewline := bytes.HasSuffix(content, []byte("\n"))
	lines := bytes.Split(bytes.TrimSuffix(content, []byte("\n")), []byte("\n"))
	out := make([][]byte, 0, len(lines))
	i := 0
	for n := 0; i < len(lines) && isBlank(lines[i]); i++ {
		if n < l.leading {
			out = append(out, lines[i])
			n++
		}
	}
	run := 0
	for ; i < len(lines); i++ {
		if isBlank(lines[i]) {
			run++
			if run > maxRun {
				continue
			}
		} else {
			run = 0
		}
		out = append(out, lines[i])
	}
	result := bytes.Join(out, []byte("\n"))
	if trailingNewline {
		result = append(result, '\n')
	}
	return result
}
