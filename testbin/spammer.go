package main

import (
	"fmt"
	"os"
	"strings"
)

func spammer(lable string, nonewline bool, target *os.File) {
	spam := newSpamCan(10, lable, nonewline)
	for {
		target.WriteString(spam.getSpam())
	}
}

type spamCan struct {
	offset        int
	spamIncrement int
	currentSpam   []string
	endLineToken  string
	lable         string
}

func newSpamCan(increment int, lable string, noNewLine bool) *spamCan {
	sc := &spamCan{
		offset:        33,
		currentSpam:   make([]string, 0),
		spamIncrement: increment,
		lable:         lable,
		endLineToken:  "",
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
	for i := 0; i < spam.spamIncrement; i++ {
		nextChar := string(byte(spam.nextOffset()))
		spam.currentSpam = append(spam.currentSpam, nextChar)
	}
	return fmt.Sprintf("%s - %s%s", spam.lable, strings.Join(spam.currentSpam, ""), spam.endLineToken)
}
