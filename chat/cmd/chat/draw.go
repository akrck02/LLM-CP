package main

import (
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/akrck02/go-renderer/draw2d"
	ui "github.com/akrck02/go-ui"

	"github.com/akrck02/LLM-CP/chat/conversation"
)

// Measures of the chat, in pixels.
const (
	chatListWidth   = 340
	listAvatar      = 44
	messageAvatar   = 34
	bubbleRadius    = 14
	bubbleMaxWidth  = 520
	bubbleShare     = 0.72 // of the page's width, on a narrow screen
	letterWidth     = 9    // about, in pixels: a line of more letters than fit is wrapped
	previewLetters  = 24
	paneGap         = 12
	narrowWidth     = 720 // under this, a phone: one pane at a time
	membersLetters  = 40
	paneRadius      = 18
	setupWidth      = 360
	messagesPadding = 16
	messageGap      = 8
	markSize        = 18  // the dot or sign over a picture's corner
	groupRoundness  = 0.3 // of a group picture's size, its corners' radius
	faceSize        = 30
)

// nameColours are the colours sessions get, for their pictures and names: one each, always the
// same. Pastels, so the many pictures of a list stay quiet beside the text.
var nameColours = []draw2d.Color{
	draw2d.MustHex("#F2B8BC"), draw2d.MustHex("#BFE3B0"), draw2d.MustHex("#F3E0A8"),
	draw2d.MustHex("#B3D4EE"), draw2d.MustHex("#D2C8F2"), draw2d.MustHex("#F4C2DA"),
	draw2d.MustHex("#B2E2E0"), draw2d.MustHex("#F8CEB0"),
}

// initialsInk writes the initials on a pastel picture: a soft dark, as white would barely show.
var initialsInk = draw2d.MustHex("#2A2C33")

// nameInkShare is how much of the page's ink a name written in its colour takes: a pastel alone
// would vanish on a light page.
const nameInkShare = 0.35

// nameInk is the colour a name is written in: its pastel, leaning toward the page's ink.
func nameInk(theme ui.Theme, name string) draw2d.Color {
	colour, ink := colourOf(name), theme.Ink
	var mixed draw2d.Color
	for channel := range mixed {
		mixed[channel] = colour[channel] + (ink[channel]-colour[channel])*nameInkShare
	}
	return mixed
}

// besideBubble is the room a message from another session needs beside its text on a phone: the
// panes' gaps, the messages' padding, the sender's picture with its gap, and the bubble's padding.
// Without it the widest bubbles ran past the pane's edge and were cut.
const besideBubble = 2*paneGap + 2*messagesSidePadding + messageAvatar + 8 + 2*bubbleSidePadding

// messagesSidePadding and bubbleSidePadding keep the messages off the pane's sides and the text off
// the bubble's.
const (
	messagesSidePadding = 24
	bubbleSidePadding   = 12
)

func colourOf(name string) draw2d.Color {
	return nameColours[conversation.ColorIndex(name, len(nameColours))]
}

// draw lays out the whole page: glass's ground, and over it the chats on the left and the open
// one beside them, as panes.
func (chat *screen) draw(context *ui.Context) {
	chat.mutex.Lock()
	defer chat.mutex.Unlock()
	width, height := context.Size()
	context.Paint(ui.Layout{Width: ui.Grow(), Height: ui.Grow()}, func(area draw2d.Rectangle, drawing *draw2d.List) {
		chat.paintGround(area, drawing)
		if width < narrowWidth {
			drawing.Rectangle(area, 0, context.Theme.Surface, draw2d.Color{}, 0)
		}
	})
	narrow := width < narrowWidth
	chat.narrow = narrow
	chat.bubbleWidth = min(bubbleMaxWidth, width*bubbleShare, width-besideBubble)
	context.Overlay("screen", ui.LayerOptions{Width: ui.Fixed(width), Height: ui.Fixed(height)}, func() {
		insets := context.Insets() // a phone's status and gesture bars; zero in a browser
		gap := float32(paneGap)
		if narrow {
			gap = 0 // a phone gives the whole screen to the one pane, short of its bars
		}
		clear := ui.Padding{Top: insets.Top + gap, Bottom: insets.Bottom + gap, Left: insets.Left + gap, Right: insets.Right + gap}
		context.Box(ui.Layout{Width: ui.Grow(), Height: ui.Grow(), Padding: clear, Gap: paneGap}, func() {
			if chat.drawSetup(context) {
				return
			}
			switch {
			case !narrow:
				chat.chatList(context, ui.Fixed(chatListWidth))
				chat.openChat(context, false)
			case chat.showingList || chat.open == "":
				chat.chatList(context, ui.Grow())
			default:
				chat.openChat(context, true)
			}
		})
	})
	chat.groupDialog(context)
	// the phone's back goes from an open chat to the list, after the dialog had its chance to
	// close; on the list it is left unanswered, and the app leaves
	if narrow && !chat.showingList && context.Back() {
		chat.showingList = true
	}
	chat.keepReading()
	chat.afterFrame(narrow)
}

// chatList is the side pane: its header, the sections as tabs (at the bottom on a phone) and the
// chosen section's list.
func (chat *screen) chatList(context *ui.Context, width ui.Sizing) {
	theme := context.Theme
	narrow := width == ui.Grow()
	pane := chat.paneLook(theme, true)
	if narrow {
		pane.ShiftX = -(1 - context.Appear("list", chat.motion(paneSlide))) * slideDistance
	}
	context.Frame(ui.Layout{Direction: ui.Vertical, Width: width, Height: ui.Grow()}, pane, func() {
		chat.listHeader(context)
		if !narrow {
			chat.tabs(context)
		}
		chat.sectionContent(context)
		if narrow {
			chat.bottomNavigation(context)
		}
	})
}

// listHeader names the app, says whether it is live, and offers to make a group.
func (chat *screen) listHeader(context *ui.Context) {
	theme := context.Theme
	context.Box(ui.Layout{Width: ui.Grow(), Padding: ui.Padding{Top: 10, Bottom: 4, Left: 16, Right: 8}, Gap: 10, AlignY: ui.Center}, func() {
		chat.logo(context)
		context.Text("LLM-CP", ui.TextStyle{Size: theme.HeadingSize, Weight: draw2d.Bold})
		status, colour := "connecting…", theme.InkSoft
		if chat.online {
			status, colour = "live", theme.Primary
			chat.liveDot(context)
		}
		context.Text(status, ui.TextStyle{Size: theme.SmallTextSize, Color: colour, Weight: draw2d.Bold})
		context.Spacer()
		if context.FlatIconButton("user-plus", "") {
			chat.creating = newGroup{open: true}
		}
	})
}

// liveDot is the small dot before "live" while the page is connected. It stays still: a pulse
// redrew the whole screen thirty times a second, for ever, which made phones stutter.
func (chat *screen) liveDot(context *ui.Context) {
	const size = 8
	look := ui.Look{Fill: context.Theme.Primary, Radius: size / 2}
	context.Frame(ui.Layout{Width: ui.Fixed(size), Height: ui.Fixed(size)}, look, func() {})
}

// openTopic opens a chat, on a phone in place of the list.
func (chat *screen) openTopic(topic string) {
	chat.open, chat.showingList = topic, false
	chat.section = sessionsSection
	if !strings.HasPrefix(topic, "@") {
		chat.section = topicsSection // the list shows the chat opened among its kind
	}
}

// chatRow is one chat in the list; it tells whether it was clicked. A chat being archived slides
// out and fades, and is archived once it is gone.
func (chat *screen) chatRow(context *ui.Context, row *conversation.Chat) bool {
	theme := context.Theme
	looks := rowLooks(theme, row.Topic == chat.open)
	if row.Topic == chat.leaving {
		gone := context.Appear("leave "+row.Topic, chat.motion(rowLeave))
		for _, look := range []*ui.Look{&looks.Normal, &looks.Hovered, &looks.Pressed} {
			look.Fade, look.ShiftX = gone, -gone*leaveDistance
		}
		if gone >= 1 {
			chat.finishArchiving(row.Topic)
		}
	}
	layout := ui.Layout{Width: ui.Grow(), Padding: ui.PadXY(10, 8), Gap: 12, AlignY: ui.Center}
	return context.Clickable("chat "+row.Topic, layout, looks, func() {
		chat.chatAvatar(context, row.Topic, listAvatar)
		context.Box(ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Gap: 4}, func() {
			last, written := row.Last()
			context.Box(ui.Layout{Width: ui.Grow(), AlignY: ui.Center}, func() {
				context.Text(conversation.Title(row.Topic), ui.TextStyle{Weight: draw2d.Bold})
				if chat.log.Group(row.Topic) != nil {
					context.Chip("", "group", ui.ChipLook{Fill: theme.PrimaryTint, Ink: theme.Primary})
				}
				context.Spacer()
				if written {
					context.Text(last.Published.Format("15:04"), ui.TextStyle{Size: theme.SmallTextSize, Color: theme.InkSoft})
				}
			})
			context.Box(ui.Layout{Width: ui.Grow(), AlignY: ui.Center, Gap: 6}, func() {
				if typing := chat.log.TypingIn(row.Topic, time.Now()); len(typing) > 0 {
					chat.typingLine(context, typingText(typing), time.Now())
				} else if chat.log.IsMetrics(row.Topic) {
					context.Text(seriesCount(chat.log.SeriesIn(row.Topic)), ui.TextStyle{Color: theme.InkSoft})
				} else if !written {
					context.Text("No messages yet", ui.TextStyle{Color: theme.InkQuiet})
				} else {
					// cut at the badge rather than push it out of the row
					context.Frame(ui.Layout{Width: ui.Grow(), Gap: 6}, ui.Look{Clip: true}, func() {
						context.Text(last.From+":", ui.TextStyle{Color: nameInk(theme, last.From)})
						context.Text(shortened(last.Text, previewLetters), ui.TextStyle{Color: theme.InkSoft})
					})
				}
				if row.Unread > 0 && row.Topic != chat.open && !chat.log.IsMetrics(row.Topic) {
					unreadBadge(context, row.Unread)
				}
			})
		})
	})
}

// openChat is the open chat: its header, and its messages over a sunken ground, newest at the bottom.
func (chat *screen) openChat(context *ui.Context, narrow bool) {
	theme := context.Theme
	open := chat.log.Open(chat.open)
	if open == nil && chat.open != "" {
		open = &conversation.Chat{Topic: chat.open} // a group just made, nobody wrote there yet
	}
	pane := chat.paneLook(theme, true)
	if narrow {
		pane.ShiftX = (1 - context.Appear("open "+chat.open, chat.motion(paneSlide))) * slideDistance
	}
	context.Frame(ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Height: ui.Grow()}, pane, func() {
		if open == nil {
			context.Box(ui.Layout{Width: ui.Grow(), Height: ui.Grow(), AlignX: ui.Center, AlignY: ui.Center}, func() {
				context.Text("Waiting for the sessions to say something…", ui.TextStyle{Color: theme.InkSoft})
			})
			return
		}
		chat.chatHeader(context, open, narrow)
		if chat.log.IsMetrics(open.Topic) {
			chat.metricsView(context, open.Topic, narrow)
			return
		}
		messagesArea := ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Height: ui.Grow(), Padding: ui.PadXY(messagesSidePadding, messagesPadding), Gap: messageGap, AlignY: ui.End}
		context.Frame(ui.Layout{Width: ui.Grow(), Height: ui.Grow()}, ui.Look{}, func() {
			context.ScrollAreaFromEnd("messages "+open.Topic, messagesArea, func() {
				messages, hidden := chat.shownMessages(open)
				if hidden > 0 && context.Button("Show earlier messages · "+strconv.Itoa(hidden)) {
					chat.shownCount[open.Topic] = len(messages) + min(hidden, messagesPerPage)
				}
				for index, message := range messages {
					firstOfRun := index == 0 || messages[index-1].From != message.From
					chat.messageRow(context, message, firstOfRun)
				}
				for _, name := range chat.log.TypingIn(open.Topic, time.Now()) {
					chat.typingBubble(context, name)
				}
			})
		})
		chat.composer(context)
	})
}

// chatHeader names the open chat, says what it is about or who is typing there, shows its
// members, and offers to subscribe to it and to archive it; on a narrow screen it starts with a
// button back to the list.
func (chat *screen) chatHeader(context *ui.Context, open *conversation.Chat, narrow bool) {
	theme := context.Theme
	context.Box(ui.Layout{Width: ui.Grow(), Padding: ui.PadXY(16, 12), Gap: 12, AlignY: ui.Center}, func() {
		if narrow && context.FlatIconButton("chevron-left", "") {
			chat.showingList = true
		}
		const size float32 = messageAvatar + 8
		radius := size / 2
		var mark func() // none, unless the chat is a group
		if chat.log.Group(open.Topic) != nil {
			radius, mark = size*groupRoundness, func() { groupMark(context) }
		}
		if chat.log.IsMetrics(open.Topic) {
			metricsAvatar(context, size)
		} else {
			markedAvatar(context, open.Topic, size, radius, mark)
		}
		context.Box(ui.Layout{Direction: ui.Vertical, Width: ui.Grow(), Gap: 3}, func() {
			context.Box(ui.Layout{Gap: 8, AlignY: ui.Center}, func() {
				context.Text(conversation.Title(open.Topic), ui.TextStyle{Weight: draw2d.Bold, Size: theme.TextSize + 1})
				context.Chip("", chat.kindOf(open), ui.ChipLook{Fill: theme.PrimaryTint, Ink: theme.Primary})
			})
			if typing := chat.log.TypingIn(open.Topic, time.Now()); len(typing) > 0 {
				chat.typingLine(context, typingText(typing), time.Now())
			} else {
				context.Text(shortened(chat.aboutOf(open), membersLetters), ui.TextStyle{Size: theme.SmallTextSize, Color: theme.InkSoft})
			}
		})
		if !narrow {
			memberFaces(context, chat.membersOf(open))
		}
		if group := chat.log.Group(open.Topic); group != nil && context.FlatIconButton("users", "") {
			chat.editMembers(*group)
		}
		chat.subscribeButton(context, open)
		context.Frame(ui.Layout{Width: ui.Fixed(1), Height: ui.Fixed(28)}, ui.Look{Fill: theme.Line}, func() {})
		if context.FlatIconButton("archive", "") {
			chat.leaving = open.Topic
		}
	})
}

// subscribeButton subscribes the person to the open topic or unsubscribes them; its icon says
// what a click does. Inboxes and the groups the person is in are always followed, so they have none.
func (chat *screen) subscribeButton(context *ui.Context, open *conversation.Chat) {
	if open.IsInbox() || chat.isMemberOf(open.Topic) {
		return
	}
	subscribed := slices.Contains(chat.log.Subscriptions(), open.Topic)
	icon := "bell-plus"
	if subscribed {
		icon = "bell-minus"
	}
	if context.FlatIconButton(icon, "") {
		chat.setSubscribed(open.Topic, !subscribed)
	}
}

// isMemberOf tells whether the person is a member of a topic's group; false when it is not one.
func (chat *screen) isMemberOf(topic string) bool {
	group := chat.log.Group(topic)
	return group != nil && slices.Contains(group.Members, chat.me)
}

// kindOf says what a chat is: a group, a session's inbox or a plain topic.
func (chat *screen) kindOf(open *conversation.Chat) string {
	switch {
	case chat.log.Group(open.Topic) != nil:
		return "group"
	case open.IsInbox():
		return "inbox"
	case chat.log.IsMetrics(open.Topic):
		return "metrics"
	}
	return "topic"
}

// aboutOf is the line under a chat's name: a group's theme, whose inbox it is, who publishes its
// metrics, or who listens.
func (chat *screen) aboutOf(open *conversation.Chat) string {
	if group := chat.log.Group(open.Topic); group != nil {
		if group.Theme == "" {
			return strings.Join(group.Members, ", ")
		}
		return group.Theme
	}
	if open.IsInbox() {
		return "Messages to " + strings.TrimPrefix(open.Topic, "@") + " alone"
	}
	if chat.log.IsMetrics(open.Topic) {
		return "Published by " + strings.Join(chat.log.PublishersOf(open.Topic), ", ")
	}
	listeners := chat.listenersOf(open.Topic)
	if len(listeners) == 0 {
		return "Nobody listens here yet"
	}
	return "Listening: " + strings.Join(listeners, ", ")
}

// listenersOf are the sessions on the hub that listen to a topic, by name.
func (chat *screen) listenersOf(topic string) []string {
	var listeners []string
	for _, contact := range chat.log.Contacts() {
		if slices.Contains(contact.Topics, topic) {
			listeners = append(listeners, contact.Name)
		}
	}
	return listeners
}

// membersOf are a group's members, or those who wrote in any other chat.
func (chat *screen) membersOf(open *conversation.Chat) []string {
	if group := chat.log.Group(open.Topic); group != nil {
		return group.Members
	}
	return open.Members
}

// memberFaces are small pictures of the members, overlapping, as a group's header shows them.
func memberFaces(context *ui.Context, members []string) {
	const shown, overlap = 5, 8
	context.Box(ui.Layout{AlignY: ui.Center}, func() {
		for index, member := range members[:min(len(members), shown)] {
			look := ui.Look{Fill: colourOf(member), Radius: faceSize / 2, Border: context.Theme.Surface, BorderWidth: 2, ShiftX: -float32(index) * overlap}
			context.Frame(ui.Layout{Width: ui.Fixed(faceSize), Height: ui.Fixed(faceSize), AlignX: ui.Center, AlignY: ui.Center}, look, func() {
				context.Text(conversation.Initials(member), ui.TextStyle{Size: faceSize * 0.36, Weight: draw2d.Bold, Color: initialsInk})
			})
		}
	})
}

// typingBubble is the bubble with hopping dots that stands for an answer on its way.
func (chat *screen) typingBubble(context *ui.Context, name string) {
	theme := context.Theme
	appeared := context.Appear("typing "+name, chat.motion(messageRise))
	look := rising(appeared, ui.Look{})
	context.Frame(ui.Layout{Width: ui.Grow(), Gap: 8, AlignY: ui.End}, look, func() {
		avatar(context, name, messageAvatar)
		context.Frame(ui.Layout{Padding: ui.PadXY(14, 12)}, ui.Look{Fill: theme.Sunken, Radius: bubbleRadius}, func() {
			chat.typingDots(context, bubbleDotSize, theme.InkSoft, time.Now())
		})
	})
}

// finishArchiving archives a chat once it has slid out of the list, and opens the next one.
func (chat *screen) finishArchiving(topic string) {
	chat.leaving = ""
	chat.act("archive", conversation.Action{As: chat.me, Topic: topic})
	chat.log.SetState(chat.log.WithArchived(topic)) // at once, without waiting for the hub's answer
	chat.showingList = true
	if chats := chat.log.Chats(); len(chats) > 0 {
		chat.open = chats[0].Topic
	}
}

// messageRow is one message: on the left with its sender's picture, or on the right when it is
// one's own (?as=). Only the first of a run from the same sender shows the picture and the name.
func (chat *screen) messageRow(context *ui.Context, message conversation.Message, firstOfRun bool) {
	own := message.From == chat.me
	alignment := ui.Start
	if own {
		alignment = ui.End
	}
	appeared := context.Appear("message "+strconv.FormatUint(message.Number, 10), chat.motion(messageRise))
	context.Frame(ui.Layout{Width: ui.Grow(), Gap: 8, AlignX: alignment, AlignY: ui.End}, rising(appeared, ui.Look{}), func() {
		if !own {
			if firstOfRun {
				avatar(context, message.From, messageAvatar)
			} else {
				context.Box(ui.Layout{Width: ui.Fixed(messageAvatar)}, func() {})
			}
		}
		messageBubble(context, message, own, firstOfRun && !own, chat.bubbleWidth)
	})
}

// messageBubble is the rounded box with the sender's name, the text and the time.
func messageBubble(context *ui.Context, message conversation.Message, own, named bool, widest float32) {
	theme := context.Theme
	look := ui.Look{Fill: theme.Sunken, Radius: bubbleRadius}
	if own {
		look.Fill = theme.PrimaryTint
	}
	context.Frame(ui.Layout{Direction: ui.Vertical, Padding: ui.PadXY(bubbleSidePadding, 8), Gap: 4}, look, func() {
		if named {
			context.Text(message.From, ui.TextStyle{Size: theme.SmallTextSize, Weight: draw2d.Bold, Color: nameInk(theme, message.From)})
		}
		for _, line := range strings.Split(message.Text, "\n") {
			wrappedLine(context, line, widest)
		}
		context.Box(ui.Layout{Width: ui.Grow(), AlignX: ui.End}, func() {
			context.Text("#"+strconv.FormatUint(message.Number, 10)+" · "+message.Published.Format("15:04"), ui.TextStyle{Size: theme.SmallTextSize, Color: theme.InkSoft})
		})
	})
}

// wrappedLine writes a line of a message; a long one breaks into lines at the bubble's widest.
func wrappedLine(context *ui.Context, line string, widest float32) {
	if float32(len([]rune(line)))*letterWidth <= widest {
		context.Text(line, ui.TextStyle{})
		return
	}
	context.Box(ui.Layout{Width: ui.Fixed(widest), Gap: 4, LineGap: 2, Wrap: true}, func() {
		for _, word := range strings.Fields(line) {
			for _, piece := range wordPieces(word, int(widest/letterWidth)) {
				context.Text(piece, ui.TextStyle{})
			}
		}
	})
}

// avatar is a round picture with a name's initials, in the name's colour.
func avatar(context *ui.Context, name string, size float32) {
	markedAvatar(context, name, size, size/2, nil)
}

// chatAvatar is a chat's picture: round for a session or a topic, a rounded square for a group.
func (chat *screen) chatAvatar(context *ui.Context, topic string, size float32) {
	radius := size / 2
	if chat.log.Group(topic) != nil {
		radius = size * groupRoundness
	}
	if chat.log.IsMetrics(topic) {
		metricsAvatar(context, size)
		return
	}
	markedAvatar(context, topic, size, radius, nil)
}

// markedAvatar is a picture with a name's initials and, when mark is given, a small mark drawn
// over its bottom right corner (a dot that says someone is there, a group's sign).
func markedAvatar(context *ui.Context, name string, size, radius float32, mark func()) {
	look := ui.Look{Fill: colourOf(name), Radius: radius}
	layout := ui.Layout{Direction: ui.Vertical, Width: ui.Fixed(size), Height: ui.Fixed(size), AlignX: ui.Center, AlignY: ui.Center}
	initials := ui.TextStyle{Size: size * 0.38, Weight: draw2d.Bold, Color: initialsInk}
	context.Frame(layout, look, func() {
		if mark == nil {
			context.Text(conversation.Initials(name), initials)
			return
		}
		// the initials and the mark are stacked in the middle; each is shifted to its place, so
		// the initials stay centred and the mark sits on the corner
		context.Frame(ui.Layout{}, ui.Look{ShiftY: markSize / 2}, func() {
			context.Text(conversation.Initials(name), initials)
		})
		context.Frame(ui.Layout{}, ui.Look{ShiftX: size/2 - markSize/2 + 2, ShiftY: size/2 - markSize - initials.Size/2 + 6}, mark)
	})
}

// presenceDot is the green dot over a contact's picture that says it is on the hub.
func presenceDot(context *ui.Context) {
	theme := context.Theme
	look := ui.Look{Fill: theme.Positive, Radius: markSize / 2, Border: theme.Surface, BorderWidth: 3}
	context.Frame(ui.Layout{Width: ui.Fixed(markSize), Height: ui.Fixed(markSize)}, look, func() {})
}

// groupMark is the sign over a group's picture: a small round with the Lucide people icon.
func groupMark(context *ui.Context) {
	theme := context.Theme
	look := ui.Look{Fill: theme.Surface, Radius: markSize / 2}
	context.Frame(ui.Layout{Width: ui.Fixed(markSize), Height: ui.Fixed(markSize), AlignX: ui.Center, AlignY: ui.Center}, look, func() {
		context.IconSized("users", markSize-6, theme.Primary)
	})
}

// unreadBadge is the round count of unread messages.
func unreadBadge(context *ui.Context, count int) {
	theme := context.Theme
	look := ui.Look{Fill: theme.Primary, Radius: 10}
	context.Frame(ui.Layout{Height: ui.Fixed(20), Padding: ui.PadXY(7, 0), AlignY: ui.Center}, look, func() {
		context.Text(strconv.Itoa(count), ui.TextStyle{Size: theme.SmallTextSize, Weight: draw2d.Bold, Color: theme.OnPrimary})
	})
}

// shortened is a text cut to a number of letters, on one line, with an ellipsis if it was cut.
func shortened(text string, letters int) string {
	runes := []rune(strings.ReplaceAll(text, "\n", " "))
	if len(runes) <= letters {
		return string(runes)
	}
	return string(runes[:letters]) + "…"
}

// paneLook is the look of a pane: a rounded card floating on the ground. On a phone the pane is
// the whole screen, so the ground carries its fill instead (paintGround), under the system's bars too.
func (chat *screen) paneLook(theme ui.Theme, clip bool) ui.Look {
	if chat.narrow {
		return ui.Look{Clip: clip}
	}
	return ui.Look{Fill: theme.Surface, Radius: paneRadius, Clip: clip}
}

// messagesPerPage is how many of a chat's last messages are drawn, and how many more each "show
// earlier" adds: every message drawn is laid out again in every frame, so a long chat drawn whole
// made phones stutter.
const messagesPerPage = 60

// shownMessages are the chat's last messages to draw, and how many earlier ones are left out.
func (chat *screen) shownMessages(open *conversation.Chat) ([]conversation.Message, int) {
	count := max(chat.shownCount[open.Topic], messagesPerPage)
	return lastOf(open.Messages, count)
}

// lastOf is the last count messages, and how many came before them.
func lastOf(messages []conversation.Message, count int) ([]conversation.Message, int) {
	if len(messages) <= count {
		return messages, 0
	}
	return messages[len(messages)-count:], len(messages) - count
}
