package safety

import "testing"

// TestRecover_StopsAPanicFromPropagating is the whole point of this
// package: without Recover, this test's panic would crash the entire test
// binary (matching how, in production, an unrecovered panic in any
// goroutine crashes the whole server) rather than simply failing this one
// test function.
func TestRecover_StopsAPanicFromPropagating(t *testing.T) {
	ranAfterPanic := false

	func() {
		defer Recover("test_job")
		panic("boom")
	}()

	ranAfterPanic = true
	if !ranAfterPanic {
		t.Fatal("execution did not resume after the panic — Recover failed to stop it")
	}
}
