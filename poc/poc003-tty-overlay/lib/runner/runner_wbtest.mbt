///|
test "strip_trailing_newline: single line with newline" {
  inspect(strip_trailing_newline("foo\n"), content="foo")
}

///|
test "strip_trailing_newline: multi-line with newline" {
  inspect(strip_trailing_newline("a\nb\n"), content="a\nb")
}

///|
test "strip_trailing_newline: multiple trailing newlines removes only last" {
  inspect(strip_trailing_newline("a\n\n\n"), content="a\n\n")
}

///|
test "strip_trailing_newline: empty string" {
  inspect(strip_trailing_newline(""), content="")
}

///|
test "strip_trailing_newline: no trailing newline" {
  inspect(strip_trailing_newline("foo"), content="foo")
}

///|
test "strip_trailing_newline: newline only" {
  inspect(strip_trailing_newline("\n"), content="")
}
