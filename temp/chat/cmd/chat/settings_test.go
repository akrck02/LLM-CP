package main

import (
	"testing"

	"github.com/akrck02/bubble"
)

// A person who never chose sees the chat as their system is; a choice in the settings wins.
func TestTheChatOpensInTheSystemsModeUntilThePersonChooses(test *testing.T) {
	cases := []struct {
		kept         string
		systemIsDark bool
		want         bubble.Mode
	}{
		{"", false, bubble.ModeLight},
		{"", true, bubble.ModeDark},
		{lightMode, true, bubble.ModeLight},
		{darkMode, false, bubble.ModeDark},
	}
	for _, each := range cases {
		if got := startingMode(each.kept, each.systemIsDark); got != each.want {
			test.Errorf("kept %q, dark system %v: %s, want %s", each.kept, each.systemIsDark, got, each.want)
		}
	}
}

// When the system turns from day to night the chat follows it, unless the person chose a mode.
func TestTheSystemsChangeIsFollowedOnlyWithoutAChoice(test *testing.T) {
	chat := &screen{}
	chat.followSystemMode(false)
	if chat.mode != bubble.ModeLight {
		test.Fatalf("a light system gave %s", chat.mode)
	}
	chat.followSystemMode(true)
	if chat.mode != bubble.ModeDark {
		test.Fatalf("the system turned dark and the chat stayed %s", chat.mode)
	}
	chat.modeChosen = true
	chat.followSystemMode(false)
	if chat.mode != bubble.ModeDark {
		test.Fatal("the system's change overrode the person's choice")
	}
}
