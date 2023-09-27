package main

import (
	"regexp"
	"strings"
	"time"
)

func Filter[T any](in <-chan T, filterFunctions ...func(T) bool) <-chan T {
	out := make(chan T, cap(in))
	go func() {
		defer close(out)
	ElementLoop:
		for element := range in {
			for _, want := range filterFunctions {
				if !want(element) {
					continue ElementLoop
				}
			}
			out <- element
		}
	}()
	return out
}

func CreatedBefore[T Dater](t time.Time) func(T) bool {
	return func(resource T) bool {
		if resource.CreatedAt().Before(t) {
			return true
		}
		return false
	}
}

func IDIsNot[T Identifier](ids ...string) func(T) bool {
	return func(resource T) bool {
		for i := range ids {
			if resource.ID() == ids[i] {
				return false
			}
		}
		return true
	}
}

func NameIsNot[T Namer](names ...string) func(T) bool {
	return func(resource T) bool {
		for i := range names {
			if resource.Name() == names[i] {
				return false
			}
		}
		return true
	}
}

func NameDoesNotContain[T Namer](substrings ...string) func(T) bool {
	return func(resource T) bool {
		for i := range substrings {
			if strings.Contains(resource.Name(), substrings[i]) {
				return false
			}
		}
		return true
	}
}

func NameMatchesOneOfThesePatterns[T Namer](regexps ...string) func(T) bool {
	patterns := make([]*regexp.Regexp, len(regexps))
	for i := range regexps {
		patterns[i] = regexp.MustCompile(regexps[i])
	}

	return func(resource T) bool {
		for i := range patterns {
			if patterns[i].MatchString(resource.Name()) {
				return true
			}
		}
		return false
	}
}

func TagsDoNotContain(tag string) func(Resource) bool {
	return func(resource Resource) bool {
		if tagger, ok := resource.(Tagger); ok {
			for _, have := range tagger.Tags() {
				if have == tag {
					return false
				}
			}
		}
		return true
	}
}
