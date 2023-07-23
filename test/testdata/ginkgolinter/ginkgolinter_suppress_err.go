//golangcitest:config_path configs/ginkgolinter_suppress_err.yml
//golangcitest:args --disable-all -Eginkgolinter
package ginkgolinter

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func LenUsecase_err() {
	var fakeVarUnderTest []int
	Expect(fakeVarUnderTest).Should(BeEmpty())     // valid
	Expect(fakeVarUnderTest).ShouldNot(HaveLen(5)) // valid

	Expect(len(fakeVarUnderTest)).Should(Equal(0))           // want "ginkgo-linter: wrong length assertion; consider using .Expect\\(fakeVarUnderTest\\)\\.Should\\(BeEmpty\\(\\)\\). instead"
	Expect(len(fakeVarUnderTest)).ShouldNot(Equal(2))        // want "ginkgo-linter: wrong length assertion; consider using .Expect\\(fakeVarUnderTest\\)\\.ShouldNot\\(HaveLen\\(2\\)\\). instead"
	Expect(len(fakeVarUnderTest)).To(BeNumerically("==", 0)) // // want "ginkgo-linter: wrong length assertion; consider using .Expect\\(fakeVarUnderTest\\)\\.To\\(BeEmpty\\(\\)\\). instead"

	fakeVarUnderTest = append(fakeVarUnderTest, 3)
	Expect(len(fakeVarUnderTest)).ShouldNot(Equal(0))        // want "ginkgo-linter: wrong length assertion; consider using .Expect\\(fakeVarUnderTest\\)\\.ShouldNot\\(BeEmpty\\(\\)\\). instead"
	Expect(len(fakeVarUnderTest)).Should(Equal(1))           // want "ginkgo-linter: wrong length assertion; consider using .Expect\\(fakeVarUnderTest\\)\\.Should\\(HaveLen\\(1\\)\\). instead"
	Expect(len(fakeVarUnderTest)).To(BeNumerically(">", 0))  // want "ginkgo-linter: wrong length assertion; consider using .Expect\\(fakeVarUnderTest\\)\\.ToNot\\(BeEmpty\\(\\)\\). instead"
	Expect(len(fakeVarUnderTest)).To(BeNumerically(">=", 1)) // want "ginkgo-linter: wrong length assertion; consider using .Expect\\(fakeVarUnderTest\\)\\.ToNot\\(BeEmpty\\(\\)\\). instead"
	Expect(len(fakeVarUnderTest)).To(BeNumerically("!=", 0)) // want "ginkgo-linter: wrong length assertion; consider using .Expect\\(fakeVarUnderTest\\)\\.ToNot\\(BeEmpty\\(\\)\\). instead"
}

func NilUsecase_err() {
	y := 5
	x := &y
	Expect(x == nil).To(Equal(true)) // want "ginkgo-linter: wrong nil assertion; consider using .Expect\\(x\\)\\.To\\(BeNil\\(\\)\\). instead"
	Expect(nil == x).To(Equal(true)) // want "ginkgo-linter: wrong nil assertion; consider using .Expect\\(x\\)\\.To\\(BeNil\\(\\)\\). instead"
	Expect(x != nil).To(Equal(true)) // want "ginkgo-linter: wrong nil assertion; consider using .Expect\\(x\\)\\.ToNot\\(BeNil\\(\\)\\). instead"
	Expect(x == nil).To(BeTrue())    // want "ginkgo-linter: wrong nil assertion; consider using .Expect\\(x\\)\\.To\\(BeNil\\(\\)\\). instead"
	Expect(x == nil).To(BeFalse())   // want "ginkgo-linter: wrong nil assertion; consider using .Expect\\(x\\)\\.ToNot\\(BeNil\\(\\)\\). instead"
}
func BooleanUsecase_err() {
	x := true
	Expect(x).To(Equal(true)) // want "ginkgo-linter: wrong boolean assertion; consider using .Expect\\(x\\)\\.To\\(BeTrue\\(\\)\\). instead"
	x = false
	Expect(x).To(Equal(false)) // want "ginkgo-linter: wrong boolean assertion; consider using .Expect\\(x\\)\\.To\\(BeFalse\\(\\)\\). instead"
}

// ErrorUsecase_err should not trigger any warning
func ErrorUsecase_err() {
	err := errors.New("fake error")
	funcReturnsErr := func() error { return err }

	Expect(err).To(BeNil())
	Expect(err == nil).To(Equal(true))
	Expect(err == nil).To(BeFalse())
	Expect(err != nil).To(BeTrue())
	Expect(funcReturnsErr()).To(BeNil())
}

func HaveLen0Usecase_err() {
	x := make([]string, 0)
	Expect(x).To(HaveLen(0)) // want "ginkgo-linter: wrong length assertion; consider using .Expect\\(x\\)\\.To\\(BeEmpty\\(\\)\\). instead"
}

func WrongComparisonUsecase_err() {
	x := 8
	Expect(x == 8).To(BeTrue())    // want "ginkgo-linter: wrong comparison assertion; consider using .Expect\\(x\\)\\.To\\(Equal\\(8\\)\\). instead"
	Expect(x < 9).To(BeTrue())     // want "ginkgo-linter: wrong comparison assertion; consider using .Expect\\(x\\)\\.To\\(BeNumerically\\(\"<\", 9\\)\\). instead"
	Expect(x < 7).To(Equal(false)) // want "ginkgo-linter: wrong comparison assertion; consider using .Expect\\(x\\)\\.ToNot\\(BeNumerically\\(\"<\", 7\\)\\). instead"

	p1, p2 := &x, &x
	Expect(p1 == p2).To(Equal(true)) // want "ginkgo-linter: wrong comparison assertion; consider using .Expect\\(p1\\).To\\(BeIdenticalTo\\(p2\\)\\). instead"
}

func slowInt_err() int {
	time.Sleep(time.Second)
	return 42
}

func WrongEventuallyWithFunction_err() {
	Eventually(slowInt_err).Should(Equal(42))   // valid
	Eventually(slowInt_err()).Should(Equal(42)) // want "ginkgo-linter: use a function call in Eventually. This actually checks nothing, because Eventually receives the function returned value, instead of function itself, and this value is never changed; consider using .Eventually\\(slowInt_err\\)\\.Should\\(Equal\\(42\\)\\). instead"
}

var _ = Describe("Should warn for focused containers", func() {
	FContext("should not allow FContext", func() { // want "ginkgo-linter: Focus container found. This is used only for local debug and should not be part of the actual source code, consider to replace with \"Context\""
		FWhen("should not allow FWhen", func() { // want "ginkgo-linter: Focus container found. This is used only for local debug and should not be part of the actual source code, consider to replace with \"When\""
			FIt("should not allow FIt", func() {}) // want "ginkgo-linter: Focus container found. This is used only for local debug and should not be part of the actual source code, consider to replace with \"It\""
		})
	})
})
