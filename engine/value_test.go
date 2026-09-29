package engine

import "testing"

func TestValuesEqualDoesNotCoerceStrings(t *testing.T) {
	if valuesEqual(1.0, "1") {
		t.Fatal("number dan string tidak boleh dianggap sama")
	}
	if !valuesEqual(1, int64(1)) {
		t.Fatal("jenis numeric harus dibandingkan berdasarkan nilai")
	}
	if !valuesEqual("abc", "abc") || valuesEqual("abc", "def") {
		t.Fatal("perbandingan string salah")
	}
}
