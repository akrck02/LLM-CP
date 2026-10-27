package conversation

import (
	"maps"
	"slices"

	"github.com/akrck02/LLM-CP/hub"
)

// TopicKind is a declared topic: its kind, and who has published to it.
type TopicKind struct {
	Topic      string   `json:"topic"`
	Kind       string   `json:"kind"`
	Publishers []string `json:"publishers"`
}

// PublishersOf are who has published to a declared topic.
func (log *Log) PublishersOf(topic string) []string {
	for _, kind := range log.state.Kinds {
		if kind.Topic == topic {
			return kind.Publishers
		}
	}
	return nil
}

// IsMetrics tells whether a topic carries points instead of a conversation.
func (log *Log) IsMetrics(topic string) bool {
	return slices.ContainsFunc(log.state.Kinds, func(kind TopicKind) bool {
		return kind.Topic == topic && kind.Kind == string(hub.Metrics)
	})
}

// Series is one series of a metrics topic (a name with its labels) and its points, oldest first.
type Series struct {
	Name   string
	Points []hub.Point
}

// Latest is the series' newest point.
func (series Series) Latest() hub.Point { return series.Points[len(series.Points)-1] }

// SeriesIn are a metrics topic's series, by name, each with its points oldest first.
func (log *Log) SeriesIn(topic string) []Series {
	byName := map[string][]hub.Point{}
	if chat := log.chats[topic]; chat != nil {
		for _, message := range chat.Messages {
			point, err := hub.ParsePoint(message.Text)
			if err != nil {
				continue // the hub checks points; a text that is not one came before the topic was declared
			}
			byName[point.Series()] = append(byName[point.Series()], point)
		}
	}
	series := make([]Series, 0, len(byName))
	for _, name := range slices.Sorted(maps.Keys(byName)) {
		series = append(series, Series{Name: name, Points: byName[name]})
	}
	return series
}
