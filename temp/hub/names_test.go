package hub

import "testing"

// Sorting the names makes one conversation of two, whoever writes first.
func TestAPairsNameIsTheSameWhoeverWritesFirst(test *testing.T) {
	if PairOf("bob", "alice") != "alice+bob" || PairOf("alice", "bob") != "alice+bob" {
		test.Fatalf("PairOf gives %q and %q", PairOf("bob", "alice"), PairOf("alice", "bob"))
	}
}

// Names cannot hold a '+', so a topic with one is a pair, and only a pair.
func TestAPlusSignIsOnlyAllowedBetweenTwoNames(test *testing.T) {
	for topic, valid := range map[string]bool{
		"alice+bob": true, "api-v2+docs.site": true,
		"bob+alice": false, "alice+alice": false, "+bob": false, "alice+": false,
		"alice+bob+carol": false, "@alice+bob": false, "alice+b ob": false,
	} {
		if validTopic(topic) != valid {
			test.Errorf("%q valid: %v, want %v", topic, !valid, valid)
		}
	}
}
