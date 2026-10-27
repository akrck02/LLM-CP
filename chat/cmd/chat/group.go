package main

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"

	"github.com/akrck02/LLM-CP/chat/conversation"
)

// memberChipHeight is a member's chip in the new group dialog, in pixels: big enough to tap.
const memberChipHeight = 36

// dialogWidth is the new group dialog's width, in pixels, when the screen has room for it.
const dialogWidth = 440

// validGroupName is what the hub takes as a topic's name.
var validGroupName = regexp.MustCompile(`^[A-Za-z0-9._/-]{1,64}$`)

// newGroup is what the person is filling in the new group dialog.
type newGroup struct {
	open    bool
	name    string
	theme   string
	members []string
	// editing is whether the dialog changes an existing group's members instead of making one;
	// was is who the members were when it opened
	editing bool
	was     []string
}

// groupDialog asks for a new group's name, theme and members, picked from the contacts, and asks
// the server to make it.
func (chat *screen) groupDialog(context *ui.Context) {
	if !chat.creating.open {
		return
	}
	theme := context.Theme
	appeared := context.Appear("new group", chat.motion(dialogRise))
	options := ui.LayerOptions{AlignX: ui.Center, AlignY: ui.Center, Padding: ui.Pad(16), Hidden: 1 - appeared}
	dismissed := context.Modal("new group", options, func() {
		look := rising(appeared, ui.Look{Fill: solidSurface(theme), Radius: paneRadius, Shadow: true})
		width, _ := context.Size()
		layout := ui.Layout{Direction: ui.Vertical, Width: ui.Fixed(min(dialogWidth, width-32)), Padding: ui.Pad(24), Gap: 14}
		context.Frame(layout, look, func() { chat.groupForm(context) })
	})
	if dismissed {
		chat.creating = newGroup{}
	}
}

// groupForm is the dialog's fields, member chips and buttons.
func (chat *screen) groupForm(context *ui.Context) {
	theme := context.Theme
	if chat.creating.editing {
		context.Text("Members of "+chat.creating.name, ui.TextStyle{Size: theme.HeadingSize, Weight: draw2d.Bold})
	} else {
		context.Text("New group", ui.TextStyle{Size: theme.HeadingSize, Weight: draw2d.Bold})
		context.Text("Name", ui.TextStyle{Size: theme.SmallTextSize, Weight: draw2d.Bold, Color: theme.InkSoft})
		context.TextFieldWith("group name", &chat.creating.name, ui.TextFieldOptions{Placeholder: "release"})
		context.Text("Theme", ui.TextStyle{Size: theme.SmallTextSize, Weight: draw2d.Bold, Color: theme.InkSoft})
		context.TextFieldWith("group theme", &chat.creating.theme, ui.TextFieldOptions{Placeholder: "What this group talks about"})
	}
	context.Text("Members · "+itoaOrNone(len(chat.creating.members)), ui.TextStyle{Size: theme.SmallTextSize, Weight: draw2d.Bold, Color: theme.InkSoft})
	context.Box(ui.Layout{Width: ui.Grow(), Gap: 8, Wrap: true}, func() {
		for _, name := range chat.pickable() {
			chat.memberChip(context, name)
		}
	})
	context.Box(ui.Layout{Width: ui.Grow(), Gap: 8, AlignX: ui.End}, func() {
		if context.Button("Cancel") {
			chat.creating = newGroup{}
		}
		if chat.creating.editing {
			if context.PrimaryButton("Save") {
				chat.saveMembers()
			}
			return
		}
		if context.PrimaryButton("Create group") && chat.canCreateGroup() {
			chat.createGroup()
		}
	})
}

// memberChip is a contact the person can pick for the group, or leave out again.
func (chat *screen) memberChip(context *ui.Context, name string) {
	theme := context.Theme
	picked := slices.Contains(chat.creating.members, name)
	look := ui.Look{Fill: theme.Sunken, Radius: memberChipHeight / 2, Border: theme.Line, BorderWidth: 1}
	if picked {
		look.Fill, look.Border = theme.PrimaryTint, theme.Primary
	}
	hovered := look
	hovered.Border = theme.Primary
	layout := ui.Layout{Height: ui.Fixed(memberChipHeight), Padding: ui.Padding{Left: 5, Right: 12}, Gap: 8, AlignY: ui.Center}
	clicked := context.Clickable("member "+name, layout, ui.Looks{Normal: look, Hovered: hovered, Pressed: hovered}, func() {
		avatar(context, name, memberChipHeight-10)
		context.Text(name, ui.TextStyle{})
		if picked {
			context.IconSized("check", theme.IconSize-4, theme.Primary)
		}
	})
	if !clicked {
		return
	}
	if picked {
		chat.creating.members = slices.DeleteFunc(chat.creating.members, func(member string) bool { return member == name })
		return
	}
	chat.creating.members = append(chat.creating.members, name)
}

// canCreateGroup tells whether the dialog holds a group the hub would take.
func (chat *screen) canCreateGroup() bool {
	name := strings.TrimSpace(chat.creating.name)
	return validGroupName.MatchString(name) && len(chat.creating.members) > 0 && chat.log.Group(name) == nil
}

// createGroup asks the server for the group, opens its chat and closes the dialog.
func (chat *screen) createGroup() {
	name := strings.TrimSpace(chat.creating.name)
	chat.act("groups", conversation.Action{As: chat.me, Topic: name, Theme: strings.TrimSpace(chat.creating.theme), Members: chat.creating.members})
	chat.creating = newGroup{}
	chat.openTopic(name)
}

// itoaOrNone is a count of chosen members as a person says it.
func itoaOrNone(count int) string {
	if count == 0 {
		return "none chosen"
	}
	return strconv.Itoa(count) + " chosen"
}

// editMembers opens the dialog on a group's members, to add or take some out.
func (chat *screen) editMembers(group conversation.Group) {
	chat.creating = newGroup{open: true, editing: true, name: group.Name, theme: group.Theme,
		members: slices.Clone(group.Members), was: slices.Clone(group.Members)}
}

// pickable are the names the dialog offers: the contacts, and the members already in the group
// even when they are away, so they can be taken out too.
func (chat *screen) pickable() []string {
	var names []string
	for _, contact := range chat.log.Contacts() {
		names = append(names, contact.Name)
	}
	for _, member := range chat.creating.was {
		if !slices.Contains(names, member) {
			names = append(names, member)
		}
	}
	return names
}

// saveMembers asks the server for the members added and taken out, and closes the dialog.
func (chat *screen) saveMembers() {
	added, removed := memberChanges(chat.creating.was, chat.creating.members)
	topic := chat.creating.name
	if len(added) > 0 {
		chat.act("members/add", conversation.Action{As: chat.me, Topic: topic, Members: added})
	}
	if len(removed) > 0 {
		chat.act("members/remove", conversation.Action{As: chat.me, Topic: topic, Members: removed})
	}
	chat.creating = newGroup{}
}

// memberChanges are the names in now that were not in before, and those that left.
func memberChanges(before, now []string) (added, removed []string) {
	for _, name := range now {
		if !slices.Contains(before, name) {
			added = append(added, name)
		}
	}
	for _, name := range before {
		if !slices.Contains(now, name) {
			removed = append(removed, name)
		}
	}
	return added, removed
}
