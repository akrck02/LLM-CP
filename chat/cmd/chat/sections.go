package main

import (
	"strconv"
	"strings"
	"time"

	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"

	"github.com/akrck02/LLM-CP/chat/conversation"
)

// section is what the side pane shows: the chats, the groups, the contacts, the settings, or the
// archived chats.
type section int

const (
	sessionsSection section = iota
	topicsSection
	contactsSection
	settingsSection
	// archivedSection has no tab: it opens from a row at the top of the chats, as it is visited
	// rarely and should not take a place beside them
	archivedSection
)

// sections are the side pane's tabs, in order, with their Lucide icons.
var sections = []struct {
	label, icon string
}{
	{"Sessions", "message-circle"}, {"Topics", "users-round"}, {"Contacts", "contact"}, {"Settings", "settings"},
}

// Measures of the sections, in pixels.
const (
	tabLineHeight = 2
	navHeight     = 64
	navPillWidth  = 60
	navPillHeight = 30
	dotSize       = 6 // a typing dot in a line of text
	bubbleDotSize = 8 // a typing dot in its own bubble
)

// tabs are the sections as tabs under the side pane's header, with a line under the chosen one
// that moves to the next.
func (chat *screen) tabs(context *ui.Context) {
	theme := context.Theme
	context.Box(ui.Layout{Width: ui.Grow(), Padding: ui.PadXY(12, 0)}, func() {
		for index, each := range sections {
			chosen := section(index) == chat.section
			underline := context.Animate("tab "+each.label, boolToFloat(chosen), chat.motion(tabMove))
			ink := theme.InkSoft
			if chosen {
				ink = theme.Ink
			}
			look := ui.Look{}
			layout := ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Padding: ui.PadXY(0, 10), Gap: 8, AlignX: ui.Center}
			if context.Clickable("tab "+each.label, layout, ui.Looks{Normal: look, Hovered: look, Pressed: look}, func() {
				context.Text(chat.sectionLabel(section(index)), ui.TextStyle{Weight: draw2d.Bold, Color: ink})
				line := ui.Look{Fill: theme.Primary, Radius: 1, Fade: 1 - underline}
				context.Frame(ui.Layout{Width: ui.Grow(), Height: ui.Fixed(tabLineHeight)}, line, func() {})
			}) {
				chat.section = section(index)
			}
		}
	})
}

// bottomNavigation is the sections as a bar at the bottom of a phone's screen, each an icon
// over its name, the chosen one in a pill of the accent's tint.
func (chat *screen) bottomNavigation(context *ui.Context) {
	theme := context.Theme
	context.Box(ui.Layout{Width: ui.Grow(), Height: ui.Fixed(navHeight), Padding: ui.PadXY(8, 6)}, func() {
		for index, each := range sections {
			chosen := section(index) == chat.section
			pill := context.Animate("nav "+each.label, boolToFloat(chosen), chat.motion(tabMove))
			ink := theme.InkSoft
			if chosen {
				ink = theme.Primary
			}
			look := ui.Look{}
			layout := ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Gap: 4, AlignX: ui.Center}
			if context.Clickable("nav "+each.label, layout, ui.Looks{Normal: look, Hovered: look, Pressed: look}, func() {
				// the pill's colour fades, not the pill: Fade would take the icon with it
				pillLook := ui.Look{Fill: theme.PrimaryTint.WithAlpha(pill), Radius: navPillHeight / 2}
				pillLayout := ui.Layout{Width: ui.Fixed(navPillWidth), Height: ui.Fixed(navPillHeight), AlignX: ui.Center, AlignY: ui.Center}
				context.Frame(pillLayout, pillLook, func() { context.IconSized(each.icon, theme.IconSize, ink) })
				context.Text(chat.sectionLabel(section(index)), ui.TextStyle{Size: theme.SmallTextSize, Weight: draw2d.Bold, Color: ink})
			}) {
				chat.section = section(index)
			}
		}
	})
}

// sectionLabel is a section's name, with how many chats it holds when that helps.
func (chat *screen) sectionLabel(which section) string {
	switch which {
	case sessionsSection, topicsSection:
		if unread := chat.unreadIn(which == topicsSection); unread > 0 {
			return sections[which].label + " · " + strconv.Itoa(unread)
		}
	}
	return sections[which].label
}

// unreadIn is how many messages wait unread in the topics, or in the sessions' inboxes, archived
// ones, the open one and metrics topics apart: a point is not something to read.
func (chat *screen) unreadIn(groups bool) int {
	total := 0
	for _, each := range chat.chatsOf(groups) {
		if each.Topic != chat.open && !chat.log.IsMetrics(each.Topic) {
			total += each.Unread
		}
	}
	return total
}

// chatsOf is the chats not archived that are topics (a group is a topic with chosen members), or
// that are sessions' inboxes.
func (chat *screen) chatsOf(topics bool) []*conversation.Chat {
	var chosen []*conversation.Chat
	for _, each := range chat.log.Chats() {
		if each.IsInbox() != topics {
			chosen = append(chosen, each)
		}
	}
	return chosen
}

// sectionContent draws the chosen section's list.
func (chat *screen) sectionContent(context *ui.Context) {
	context.ScrollArea("section", ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Height: ui.Grow(), Padding: ui.Pad(6), Gap: 2}, func() {
		switch chat.section {
		case contactsSection:
			chat.contactList(context)
		case archivedSection:
			chat.archivedList(context)
		case settingsSection:
			chat.settings(context)
		case topicsSection:
			chat.archivedEntry(context)
			chat.chatRows(context, chat.chatsOf(true))
		default:
			chat.archivedEntry(context)
			chat.chatRows(context, chat.chatsOf(false))
		}
	})
}

// contactList is every session on the hub: a dot that says it is there, what it is doing, and
// the groups it is in. Clicking one opens its inbox.
func (chat *screen) contactList(context *ui.Context) {
	theme := context.Theme
	contacts := chat.log.Contacts()
	context.Box(ui.Layout{Padding: ui.PadXY(12, 6)}, func() {
		context.Text(strconv.Itoa(len(contacts))+" SESSIONS ON THE HUB", ui.TextStyle{Size: theme.SmallTextSize, Weight: draw2d.Bold, Color: theme.InkQuiet})
	})
	now := time.Now()
	for _, contact := range contacts {
		layout := ui.Layout{Width: ui.Grow(), Padding: ui.PadXY(10, 8), Gap: 12, AlignY: ui.Center}
		if context.Clickable("contact "+contact.Name, layout, rowLooks(theme, false), func() {
			chat.contactRow(context, contact, now)
		}) {
			chat.openTopic("@" + contact.Name)
		}
	}
}

// contactRow is one contact's picture, name, state and groups.
func (chat *screen) contactRow(context *ui.Context, contact conversation.Contact, now time.Time) {
	theme := context.Theme
	markedAvatar(context, contact.Name, listAvatar, listAvatar/2, func() { presenceDot(context) })
	context.Box(ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Gap: 3}, func() {
		context.Text(contact.Name, ui.TextStyle{Weight: draw2d.Bold})
		if contact.IsTyping(now) {
			chat.typingLine(context, "typing", now)
		} else {
			context.Text("online", ui.TextStyle{Size: theme.SmallTextSize, Color: theme.InkSoft})
		}
		groups := "in no group"
		if len(contact.Groups) > 0 {
			groups = "in " + strings.Join(contact.Groups, ", ")
		}
		context.Text(shortened(groups, membersLetters), ui.TextStyle{Size: theme.SmallTextSize, Color: theme.InkQuiet})
	})
}

// chatRows lists chats, each opening when clicked.
func (chat *screen) chatRows(context *ui.Context, chats []*conversation.Chat) {
	for _, each := range chats {
		if chat.chatRow(context, each) {
			chat.openTopic(each.Topic)
		}
	}
}

// archivedEntry is the row at the top of the sessions that opens the archived ones, when there are any.
func (chat *screen) archivedEntry(context *ui.Context) {
	archived := len(chat.log.Archived())
	if archived == 0 {
		return
	}
	theme := context.Theme
	look := ui.Look{Radius: 12}
	hovered := ui.Look{Fill: theme.Sunken, Radius: 12}
	layout := ui.Layout{Width: ui.Grow(), Padding: ui.PadXY(10, 8), Gap: 12, AlignY: ui.Center}
	if context.Clickable("archived entry", layout, ui.Looks{Normal: look, Hovered: hovered, Pressed: hovered}, func() {
		context.IconSized("archive", theme.IconSize, theme.InkSoft)
		context.Text("Archived", ui.TextStyle{Weight: draw2d.Bold, Color: theme.InkSoft})
		context.Spacer()
		context.Text(strconv.Itoa(archived), ui.TextStyle{Size: theme.SmallTextSize, Color: theme.InkQuiet})
	}) {
		chat.archivedFrom, chat.section = chat.section, archivedSection
	}
}

// archivedList is the archived chats, each with a button that brings it back.
func (chat *screen) archivedList(context *ui.Context) {
	theme := context.Theme
	if context.FlatIconButton("chevron-left", "Back") {
		chat.section = chat.archivedFrom
	}
	archived := chat.log.Archived()
	if len(archived) == 0 {
		context.Box(ui.Layout{Width: ui.Grow(), Padding: ui.Pad(32), AlignX: ui.Center}, func() {
			context.Text("Nothing archived. Archive a chat from its header; its messages stay.", ui.TextStyle{Color: theme.InkQuiet})
		})
		return
	}
	for _, each := range archived {
		context.Box(ui.Layout{Width: ui.Grow(), Padding: ui.PadXY(10, 8), Gap: 12, AlignY: ui.Center}, func() {
			chat.chatAvatar(context, each.Topic, listAvatar)
			context.Box(ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Gap: 2}, func() {
				context.Text(conversation.Title(each.Topic), ui.TextStyle{Weight: draw2d.Bold})
				context.Text(strconv.Itoa(len(each.Messages))+" messages", ui.TextStyle{Size: theme.SmallTextSize, Color: theme.InkQuiet})
			})
			if context.FlatIconButton("archive-restore", "Restore") {
				chat.act("restore", conversation.Action{As: chat.me, Topic: each.Topic})
				chat.openTopic(each.Topic)
			}
		})
	}
}

// typingLine is "<who> is typing" in the accent's colour, followed by the hopping dots.
func (chat *screen) typingLine(context *ui.Context, text string, now time.Time) {
	theme := context.Theme
	context.Box(ui.Layout{Gap: 6, AlignY: ui.Center}, func() {
		context.Text(text, ui.TextStyle{Size: theme.SmallTextSize, Color: theme.Primary})
		chat.typingDots(context, dotSize, theme.Primary, now)
	})
}

// typingText says who is typing: "bob is typing", "bob and carol are typing".
func typingText(names []string) string {
	if len(names) == 1 {
		return names[0] + " is typing"
	}
	return strings.Join(names, " and ") + " are typing"
}

// rowLooks are the looks of a row in a list: a wash when hovered, the accent's tint when chosen.
func rowLooks(theme ui.Theme, chosen bool) ui.Looks {
	look := ui.Look{Radius: theme.ControlRadius, Clip: true}
	hovered := look
	hovered.Fill = theme.Sunken
	if chosen {
		look.Fill, hovered.Fill = theme.PrimaryTint, theme.PrimaryTint
	}
	return ui.Looks{Normal: look, Hovered: hovered, Pressed: hovered}
}

// boolToFloat is 1 for true and 0 for false, a target for Animate.
func boolToFloat(value bool) float32 {
	if value {
		return 1
	}
	return 0
}
