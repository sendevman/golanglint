//args: -Enonamedreturns
package testdata

import "fmt"

type asdf struct {
	test string
}

func noParams() {
	return
}

var c = func() {
	return
}

var d = func() error {
	return nil
}

var e = func() (err error) { // ERROR `named return "err" with type "error" found`
	err = nil
	return
}

var (
	f = func() {
		return
	}

	g = func() error {
		return nil
	}

	h = func() (err error) { // ERROR `named return "err" with type "error" found`
		err = nil
		return
	}
)

// this should not match as the implementation does not need named parameters (see below)
type funcDefintion func(arg1, arg2 interface{}) (num int, err error)

func funcDefintionImpl(arg1, arg2 interface{}) (int, error) {
	return 0, nil
}

func funcDefintionImpl2(arg1, arg2 interface{}) (num int, err error) { // ERROR `named return "num" with type "int" found`
	return 0, nil
}

var funcVar = func() (msg string) { // ERROR `named return "msg" with type "string" found`
	msg = "c"
	return msg
}

func test() {
	a := funcVar()
	_ = a

	var function funcDefintion
	function = funcDefintionImpl
	i, err := function("", "")
	_ = i
	_ = err
	function = funcDefintionImpl2
	i, err = function("", "")
	_ = i
	_ = err
}

func good(i string) string {
	return i
}

func bad(i string, a, b int) (ret1 string, ret2 interface{}, ret3, ret4 int, ret5 asdf) { // ERROR `named return "ret1" with type "string" found`
	x := "dummy"
	return fmt.Sprintf("%s", x), nil, 1, 2, asdf{}
}

func bad2() (msg string, err error) { // ERROR `named return "msg" with type "string" found`
	msg = ""
	err = nil
	return
}

func myLog(format string, args ...interface{}) {
	return
}

type obj struct{}

func (o *obj) func1() (err error) { return nil } // ERROR `named return "err" with type "error" found`
