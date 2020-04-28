package redact

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
)

type MultiLineRedactor struct {
	re1      *regexp.Regexp
	re2      *regexp.Regexp
	maskText string
}

func NewMultiLineRedactor(re1, re2, maskText string) (*MultiLineRedactor, error) {
	compiled1, err := regexp.Compile(re1)
	if err != nil {
		return nil, err
	}
	compiled2, err := regexp.Compile(re2)
	if err != nil {
		return nil, err
	}
	return &MultiLineRedactor{re1: compiled1, re2: compiled2, maskText: maskText}, nil
}

func (r *MultiLineRedactor) Redact(input io.Reader) io.Reader {
	out, writer := io.Pipe()
	go func() {
		var err error
		defer func() {
			writer.CloseWithError(err)
		}()

		substStr := getReplacementPattern(r.re2, r.maskText)

		reader := bufio.NewReader(input)
		line1, line2, err := getNextTwoLines(reader, nil)
		if err != nil {
			// this will print 2 blank lines for empty input...
			fmt.Fprintf(writer, "%s\n", line1)
			fmt.Fprintf(writer, "%s\n", line2)
			return
		}

		flushLastLine := false
		for err == nil {
			// If line1 matches re1, then transform line2 using re2
			if !r.re1.MatchString(line1) {
				fmt.Fprintf(writer, "%s\n", line1)
				line1, line2, err = getNextTwoLines(reader, &line2)
				flushLastLine = true
				continue
			}
			flushLastLine = false

			clean := r.re2.ReplaceAllString(line2, substStr)

			// io.WriteString would be nicer, but reader strips new lines
			fmt.Fprintf(writer, "%s\n%s\n", line1, clean)
			if err != nil {
				return
			}

			line1, line2, err = getNextTwoLines(reader, nil)
		}

		if flushLastLine {
			fmt.Fprintf(writer, "%s\n", line1)
		}
	}()
	return out
}

func getNextTwoLines(reader *bufio.Reader, curLine2 *string) (line1 string, line2 string, err error) {
	line1 = ""
	line2 = ""

	if curLine2 == nil {
		line1, err = readLine(reader)
		if err != nil {
			return
		}

		line2, err = readLine(reader)
		return
	}

	line1 = *curLine2
	line2, err = readLine(reader)
	if err != nil {
		return
	}

	return
}
