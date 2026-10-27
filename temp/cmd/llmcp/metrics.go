package main

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/akrck02/LLM-CP/hub"
	"github.com/akrck02/LLM-CP/internal/failure"
	"github.com/akrck02/LLM-CP/protocol"
	"github.com/akrck02/LLM-CP/server"
)

// topic runs one of the topic commands: declare or remove.
func topic(sent call) error {
	if err := sent.given.requireArguments(2); err != nil {
		return err
	}
	action, name := sent.given.arguments[0], sent.given.arguments[1]
	switch action {
	case "declare":
		if err := sent.given.requireArguments(3); err != nil {
			return err
		}
		kind := sent.given.arguments[2]
		if err := declare(sent, name, kind); err != nil {
			return err
		}
		return sent.answer.done(name + " is a " + kind + " topic")
	case "remove":
		asked := &protocol.TopicRequest{Session: sent.session, Topic: name}
		if _, err := sent.client.RemoveTopic(sent.request, asked); err != nil {
			return err
		}
		return sent.answer.done(name + " removed, with its messages")
	}
	return failure.Wrap(UnknownCommand, "topic "+action, nil)
}

func declare(sent call, name, kind string) error {
	_, err := sent.client.Declare(sent.request, &protocol.KindRequest{Session: sent.session, Topic: name, Kind: kind})
	return err
}

// metric publishes one point to a metrics topic, declaring the topic first if nobody did.
func metric(sent call) error {
	if err := sent.given.requireArguments(3); err != nil {
		return err
	}
	name, written := sent.given.arguments[1], sent.given.arguments[2]
	point, err := pointOf(name, written, sent.given.unit, sent.given.labels)
	if err != nil {
		return err
	}
	point.Title = sent.given.title
	topicName := sent.given.arguments[0]
	if err := declare(sent, topicName, string(hub.Metrics)); err != nil && server.CodeName(err) != hub.KindTaken.Name() {
		return err
	}
	text, err := json.Marshal(point)
	if err != nil {
		return failure.Wrap(BadOption, name, err)
	}
	message, err := sent.client.Publish(sent.request, &protocol.PublishRequest{Session: sent.session, Topic: topicName, Text: string(text)})
	if err != nil {
		return err
	}
	return sent.answer.published(message)
}

// pointOf builds a point from what was typed: a value that must be a number, and labels as k=v.
func pointOf(name, written, unit string, labels []string) (hub.Point, error) {
	value, err := strconv.ParseFloat(written, 64)
	if err != nil {
		return hub.Point{}, failure.Wrap(BadOption, "the value "+written+" is not a number", err)
	}
	point := hub.Point{Name: name, Value: value, Unit: unit}
	for _, label := range labels {
		key, labelValue, found := strings.Cut(label, "=")
		if !found || key == "" {
			return hub.Point{}, failure.Wrap(BadOption, "a label is key=value: "+label, nil)
		}
		if point.Labels == nil {
			point.Labels = map[string]string{}
		}
		point.Labels[key] = labelValue
	}
	return point, nil
}
