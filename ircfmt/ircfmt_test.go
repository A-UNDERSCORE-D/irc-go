package ircfmt

import "testing"

type testcase struct {
	escaped   string
	unescaped string
}

var tests = []testcase{
	{"te$bst", "te\x02st"},
	{"te$c[green]st", "te\x033st"},
	{"te$c[red,green]st", "te\x034,3st"},
	{"te$c[green]4st", "te\x03034st"},
	{"te$c[red,green]9st", "te\x034,039st"},
	{" ▀█▄▀▪.▀  ▀ ▀  ▀ ·▀▀▀▀  ▀█▄▀ ▀▀ █▪ ▀█▄▀▪", " ▀█▄▀▪.▀  ▀ ▀  ▀ ·▀▀▀▀  ▀█▄▀ ▀▀ █▪ ▀█▄▀▪"},
	{"test $$c", "test $c"},
	{"test $c[]", "test \x03"},
	{"test $$", "test $"},
}

var escapetests = []testcase{
	{"te$c[]st", "te\x03st"},
	{"test$c[]", "test\x03"},
}

var unescapetests = []testcase{
	{"te$xt", "text"},
	{"te$st", "te\x1et"},
	{"test$c", "test\x03"},
}

var stripTests = []testcase{
	{"te\x02st", "test"},
	{"te\x033st", "test"},
	{"te\x034,3st", "test"},
	{"te\x03034st", "te4st"},
	{"te\x034,039st", "te9st"},
	{" ▀█▄▀▪.▀  ▀ ▀  ▀ ·▀▀▀▀  ▀█▄▀ ▀▀ █▪ ▀█▄▀▪", " ▀█▄▀▪.▀  ▀ ▀  ▀ ·▀▀▀▀  ▀█▄▀ ▀▀ █▪ ▀█▄▀▪"},
}

func TestEscape(t *testing.T) {
	for _, pair := range tests {
		val := Escape(pair.unescaped)

		if val != pair.escaped {
			t.Error(
				"For", pair.unescaped,
				"expected", pair.escaped,
				"got", val,
			)
		}
	}
	for _, pair := range escapetests {
		val := Escape(pair.unescaped)

		if val != pair.escaped {
			t.Error(
				"For", pair.unescaped,
				"expected", pair.escaped,
				"got", val,
			)
		}
	}
}

func TestChain(t *testing.T) {
	for _, pair := range tests {
		escaped := Escape(pair.unescaped)
		unescaped := Unescape(escaped)
		if unescaped != pair.unescaped {
			t.Errorf("for %q expected %q got %q", pair.unescaped, pair.unescaped, unescaped)
		}
	}
}

func TestUnescape(t *testing.T) {
	for _, pair := range tests {
		val := Unescape(pair.escaped)

		if val != pair.unescaped {
			t.Error(
				"For", pair.escaped,
				"expected", pair.unescaped,
				"got", val,
			)
		}
	}
	for _, pair := range unescapetests {
		val := Unescape(pair.escaped)

		if val != pair.unescaped {
			t.Error(
				"For", pair.escaped,
				"expected", pair.unescaped,
				"got", val,
			)
		}
	}
}

func TestStrip(t *testing.T) {
	for _, pair := range stripTests {
		val := Strip(pair.escaped)
		if val != pair.unescaped {
			t.Error(
				"For", pair.escaped,
				"expected", pair.unescaped,
				"got", val,
			)
		}
	}
}
