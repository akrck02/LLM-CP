package main

import (
	"slices"
	"testing"

	"github.com/akrck02/bubble"
)

func TestSavingMembersSendsOnlyWhoChanged(test *testing.T) {
	added, removed := memberChanges([]string{"alice", "bob"}, []string{"bob", "carol"})
	if !slices.Equal(added, []string{"carol"}) || !slices.Equal(removed, []string{"alice"}) {
		test.Errorf("added %v, removed %v", added, removed)
	}
}

func TestADialogHidesWhatIsUnderIt(test *testing.T) {
	chat := &screen{}
	if err := chat.useMode(bubble.ModeDark); err != nil {
		test.Fatal(err)
	}
	if fill := solidSurface(chat.theme); fill[3] != 1 {
		test.Errorf("the dialog's fill lets %v of what is under it through", 1-fill[3])
	}
}
