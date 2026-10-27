package hub

import "testing"

func TestEachErrorCodeHasItsOwnName(test *testing.T) {
	seen := map[string]bool{}
	for _, code := range Codes {
		if seen[code.Name()] {
			test.Fatalf("two codes are named %q", code.Name())
		}
		seen[code.Name()] = true
		if CodeNamed(code.Name()) != code {
			test.Fatalf("CodeNamed(%q) does not find its own code", code.Name())
		}
	}
}
