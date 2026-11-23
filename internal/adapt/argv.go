package adapt

import (
	"errors"
	"regexp"
	"strings"
)

var assignment = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)

// Argv splits a declared command the way a person reads it, without a shell. Single quotes
// keep their text as is. Double quotes keep spaces and take \" and \\ as escapes. Outside
// quotes a backslash is a plain character, so a Windows path survives. Words of the form
// NAME=value before the program are environment for that command and are returned apart.
func Argv(command string) (env []string, argv []string, err error) {
	var words []string
	var word strings.Builder
	inWord := false
	var quote rune
	runes := []rune(command)
	for index := 0; index < len(runes); index++ {
		char := runes[index]
		switch {
		case quote == '\'':
			if char == '\'' {
				quote = 0
			} else {
				word.WriteRune(char)
			}
		case quote == '"':
			switch {
			case char == '"':
				quote = 0
			case char == '\\' && index+1 < len(runes) && (runes[index+1] == '"' || runes[index+1] == '\\'):
				index++
				word.WriteRune(runes[index])
			default:
				word.WriteRune(char)
			}
		case char == '\'' || char == '"':
			quote = char
			inWord = true
		case char == ' ' || char == '\t' || char == '\n' || char == '\r':
			if inWord {
				words = append(words, word.String())
				word.Reset()
				inWord = false
			}
		default:
			word.WriteRune(char)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, nil, errors.New("command has an unterminated quote")
	}
	if inWord {
		words = append(words, word.String())
	}
	for len(words) > 0 && assignment.MatchString(words[0]) {
		env = append(env, words[0])
		words = words[1:]
	}
	if len(words) == 0 {
		return nil, nil, errors.New("command has no program")
	}
	return env, words, nil
}
