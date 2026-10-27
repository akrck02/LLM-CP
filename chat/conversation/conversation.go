// Package conversation turns the stream of messages a hub publishes into chats, one per topic,
// as a messaging app shows them: the newest chat first, its messages oldest first, with how many
// are still unread. It knows nothing of the screen or the network.
package conversation

import (
	"cmp"
	"hash/fnv"
	"maps"
	"slices"
	"strings"
	"time"
)

// Message is one message as the hub published it.
type Message struct {
	Number    uint64    `json:"number"`
	Topic     string    `json:"topic"`
	From      string    `json:"from"`
	Text      string    `json:"text"`
	Published time.Time `json:"published"`
}

// Chat is a topic and what was said in it.
type Chat struct {
	Topic    string
	Messages []Message // oldest first
	Members  []string  // who has written here, by name
	Unread   int
}

// Last is the chat's newest message, and false when nobody wrote there yet.
func (chat *Chat) Last() (Message, bool) {
	if len(chat.Messages) == 0 {
		return Message{}, false
	}
	return chat.Messages[len(chat.Messages)-1], true
}

// IsInbox tells whether the chat is a session's own topic, where others write to it alone.
func (chat *Chat) IsInbox() bool { return strings.HasPrefix(chat.Topic, "@") }

// Log holds every chat. The zero Log is empty and ready to use; it is not safe to use from two
// goroutines at once.
type Log struct {
	chats map[string]*Chat
	seen  map[uint64]bool
	state State
	// read is, per topic, the number of the newest message read there
	read map[string]uint64
}

// Add puts a message in its chat, unread, unless it was added already.
func (log *Log) Add(message Message) {
	if log.chats == nil {
		log.chats, log.seen = map[string]*Chat{}, map[uint64]bool{}
	}
	if log.seen[message.Number] {
		return
	}
	log.seen[message.Number] = true
	chat, found := log.chats[message.Topic]
	if !found {
		chat = &Chat{Topic: message.Topic}
		log.chats[message.Topic] = chat
	}
	chat.Messages = append(chat.Messages, message)
	if message.Number > log.read[message.Topic] {
		chat.Unread++
	}
	if !slices.Contains(chat.Members, message.From) {
		chat.Members = append(chat.Members, message.From)
		slices.Sort(chat.Members)
	}
}

// Chats lists the chats not archived, the one with the newest message first.
func (log *Log) Chats() []*Chat {
	return slices.DeleteFunc(log.allChats(), func(chat *Chat) bool { return log.IsArchived(chat.Topic) })
}

// allChats lists every chat, archived or not, the one with the newest message first. Groups and
// the topics sessions listen to are chats too before anyone writes there, after those that
// have messages, by name: a hub that just started is not an empty page.
func (log *Log) allChats() []*Chat {
	chats := make([]*Chat, 0, len(log.chats))
	for _, chat := range log.chats {
		chats = append(chats, chat)
	}
	for _, topic := range log.quietTopics() {
		chats = append(chats, &Chat{Topic: topic})
	}
	slices.SortFunc(chats, func(first, second *Chat) int {
		firstLast, _ := first.Last()
		secondLast, _ := second.Last()
		if order := cmp.Compare(secondLast.Number, firstLast.Number); order != 0 {
			return order
		}
		return strings.Compare(first.Topic, second.Topic)
	})
	return chats
}

// quietTopics are the groups, the declared topics and the shared topics someone listens to where
// nobody wrote yet.
// A session's own topic is left out: an inbox shows once someone writes to it.
func (log *Log) quietTopics() []string {
	var topics []string
	add := func(topic string) {
		if _, written := log.chats[topic]; !written && !strings.HasPrefix(topic, "@") && !slices.Contains(topics, topic) {
			topics = append(topics, topic)
		}
	}
	for _, group := range log.state.Groups {
		add(group.Name)
	}
	for _, kind := range log.state.Kinds {
		add(kind.Topic)
	}
	for _, contact := range log.state.Contacts {
		for _, topic := range contact.Topics {
			add(topic)
		}
	}
	return topics
}

// Open is the chat of a topic, all its messages marked as read; nil if nobody wrote there yet.
func (log *Log) Open(topic string) *Chat {
	chat := log.chats[topic]
	if chat == nil {
		return nil
	}
	chat.Unread = 0
	if last, found := chat.Last(); found && last.Number > log.read[topic] {
		if log.read == nil {
			log.read = map[string]uint64{}
		}
		log.read[topic] = last.Number
	}
	return chat
}

// ReadMarks are, per topic, the number of the newest message read there: what a person's reading
// is kept as between runs. The map is a copy.
func (log *Log) ReadMarks() map[string]uint64 { return maps.Clone(log.read) }

// SetReadMarks takes the marks kept from an earlier run, and counts again what is unread.
func (log *Log) SetReadMarks(marks map[string]uint64) {
	log.read = maps.Clone(marks)
	for topic, chat := range log.chats {
		chat.Unread = 0
		for _, message := range chat.Messages {
			if message.Number > log.read[topic] {
				chat.Unread++
			}
		}
	}
}

// Title is how a chat is named on screen: a session's inbox by the session's name, without the
// "@" the hub addresses it with ("@alice" is "alice"); a topic by its name.
func Title(topic string) string { return strings.TrimPrefix(topic, "@") }

// Initials are up to two letters that stand for a name in its round picture: "go-docs" is "GD".
func Initials(name string) string {
	words := strings.FieldsFunc(strings.TrimPrefix(name, "@"), func(character rune) bool {
		return strings.ContainsRune("-_./ ", character)
	})
	initials := ""
	for _, word := range words {
		initials += strings.ToUpper(string([]rune(word)[0]))
		if len([]rune(initials)) == 2 {
			break
		}
	}
	if initials == "" {
		return "?"
	}
	return initials
}

// ColorIndex picks one of count colours for a name, always the same for the same name, so a
// session keeps its colour in every chat.
func ColorIndex(name string, count int) int {
	hash := fnv.New32a()
	hash.Write([]byte(name)) // writing to a hash never fails
	return int(hash.Sum32() % uint32(count))
}
