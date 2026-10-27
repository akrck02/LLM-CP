package conversation

import "testing"

func TestAMetricsTopicGroupsItsPointsBySeries(test *testing.T) {
	var log Log
	log.SetState(State{Kinds: []TopicKind{{Topic: "tests", Kind: "metrics", Publishers: []string{"alice"}}}})
	for number, text := range []string{
		`{"name":"passed","value":10,"labels":{"repo":"api"}}`,
		`{"name":"passed","value":12,"labels":{"repo":"api"}}`,
		`{"name":"passed","value":3,"labels":{"repo":"docs"}}`,
	} {
		log.Add(Message{Number: uint64(number + 1), Topic: "tests", From: "alice", Text: text})
	}
	series := log.SeriesIn("tests")
	if len(series) != 2 || len(series[0].Points) != 2 || series[0].Latest().Value != 12 || series[1].Latest().Value != 3 {
		test.Errorf("the series are %+v", series)
	}
	if publishers := log.PublishersOf("tests"); len(publishers) != 1 || publishers[0] != "alice" {
		test.Errorf("the publishers are %v", publishers)
	}
	if !log.IsMetrics("tests") || log.IsMetrics("api") {
		test.Error("the topics' kinds are mixed up")
	}
}
