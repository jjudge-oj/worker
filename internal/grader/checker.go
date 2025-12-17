package grader

import (
	"fmt"
	"strings"
)

type ComparisonMode int

const (
	ComparisonModeExact ComparisonMode = iota
	ComparisonModeToken
)

type Checker struct {
	mode ComparisonMode
}

func (c *Checker) Check(out, expOut string) (bool, error) {
	switch c.mode {
	case ComparisonModeExact:
		return out == expOut, nil
	case ComparisonModeToken:
		outTokens := strings.Fields(out)
		expOutTokens := strings.Fields(expOut)
		if len(outTokens) != len(expOutTokens) {
			return false, nil
		}
		for i := range outTokens {
			if outTokens[i] != expOutTokens[i] {
				return false, nil
			}
		}
		return true, nil
	default:
		return false, fmt.Errorf("unknown comparison mode")
	}
}

type Option func(*Checker)

func WithExactComparison() Option {
	return func(c *Checker) {
		c.mode = ComparisonModeExact
	}
}

func WithTokenComparison() Option {
	return func(c *Checker) {
		c.mode = ComparisonModeToken
	}
}

func NewChecker(opts ...Option) *Checker {
	c := &Checker{
		mode: ComparisonModeExact,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}
