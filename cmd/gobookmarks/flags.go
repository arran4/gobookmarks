package main

import "strconv"

type stringFlag struct {
	value string
	set   bool
}

func (s *stringFlag) Set(v string) error { s.value = v; s.set = true; return nil }
func (s *stringFlag) String() string     { return s.value }

type boolFlag struct {
	value bool
	set   bool
}

func (b *boolFlag) Set(v string) error {
	val, err := strconv.ParseBool(v)
	if err != nil {
		return err
	}
	b.value = val
	b.set = true
	return nil
}

func (b *boolFlag) String() string { return strconv.FormatBool(b.value) }
