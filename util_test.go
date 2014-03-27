package gogsmmodem

import "fmt"

func ExampleParseTime() {
	t := parseTime("14/02/01,15:07:43+00")
	fmt.Println(t)
	// Output:
	// 2014-02-01 15:07:43 +0000 UTC
}

func ExampleStartsWith() {
	fmt.Println(startsWith("abc", "ab"))
	fmt.Println(startsWith("abc", "b"))
	// Output:
	// true
	// false
}

func ExampleQuotes() {
	args := []interface{}{"a", 1, "b"}
	fmt.Println(quotes(args))
	// Output:
	// "a",1,"b"
}

func ExampleUnquotes() {
	fmt.Println(unquotes(`"a,comma",1,"b"`))
	// Output:
	// [a,comma 1 b]
}
