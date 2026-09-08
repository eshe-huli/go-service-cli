package validate

import "testing"

func TestDecimal(t *testing.T) {
	for _, s := range []string{"0", "1.20", "-0.01", "999999999999999999999.1234"} {
		if !Decimal(s) {
			t.Fatal(s)
		}
	}
	for _, s := range []string{"NaN", "1e3", "01.2", "+1", ".1", "1.", " 1"} {
		if Decimal(s) {
			t.Fatal(s)
		}
	}
}
