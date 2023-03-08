package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/silverstagtech/randomstring"
)

func spammer(label string, size uint, noNewLine bool, target *os.File) {
	increment := 0
	if size == 0 {
		increment = 10
	}

	spam := newSpamCan(increment, label, noNewLine, size)
	for {
		_, err := target.WriteString(spam.getSpam())
		if err != nil {
			fmt.Println("Error writing to stdout:", err)
		}
	}
}

type spamCan struct {
	offset        int
	spamIncrement int
	currentSpam   []string
	endLineToken  string
	label         string
	size          uint
}

func newSpamCan(increment int, label string, noNewLine bool, size uint) *spamCan {
	sc := &spamCan{
		offset:        33,
		currentSpam:   make([]string, 0),
		spamIncrement: increment,
		label:         label,
		endLineToken:  "",
		size:          size,
	}
	if !noNewLine {
		sc.endLineToken = "\n"
	}
	return sc
}

func (spam *spamCan) nextOffset() int {
	if spam.offset < 125 {
		spam.offset++
	} else {
		spam.offset = 33
	}

	return spam.offset
}

func (spam *spamCan) getSpam() string {
	if spam.size == 0 {
		for i := 0; i < spam.spamIncrement; i++ {
			nextChar := string(byte(spam.nextOffset()))
			spam.currentSpam = append(spam.currentSpam, nextChar)
		}
		return fmt.Sprintf("%s - %s%s", spam.label, strings.Join(spam.currentSpam, ""), spam.endLineToken)
	} else {
		rs, _ := randomstring.Generate(4, 4, 4, 4, int(spam.size))
		return fmt.Sprintf("%s - %s%s",
			spam.label,
			rs,
			spam.endLineToken,
		)
	}
}
