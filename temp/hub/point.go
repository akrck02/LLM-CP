package hub

import (
	"encoding/json"
	"maps"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/akrck02/LLM-CP/internal/failure"
)

// Point is one measurement on a metrics topic, written as JSON in a message's text.
type Point struct {
	Name   string            `json:"name"`             // what is measured, like "tests.seconds"
	Title  string            `json:"title,omitempty"`  // the name as people read it, like "Test time"
	Value  float64           `json:"value"`            // the measurement
	Unit   string            `json:"unit,omitempty"`   // like "s" or "tokens"
	Labels map[string]string `json:"labels,omitempty"` // what it is about, like repository=api
	At     int64             `json:"at,omitempty"`     // when, in Unix milliseconds
}

// Series names the point's series: its name and its labels, like "tests.passed repo=api".
func (point Point) Series() string {
	parts := []string{point.Name}
	for _, key := range slices.Sorted(maps.Keys(point.Labels)) {
		parts = append(parts, key+"="+point.Labels[key])
	}
	return strings.Join(parts, " ")
}

// Shown is the point's series as people read it: its title and its labels' values, like
// "Tests passed · api"; with no title, the series' name.
func (point Point) Shown() string {
	if point.Title == "" {
		return point.Series()
	}
	parts := []string{point.Title}
	for _, key := range slices.Sorted(maps.Keys(point.Labels)) {
		parts = append(parts, point.Labels[key])
	}
	return strings.Join(parts, " · ")
}

// PointsKept is how long a metrics topic keeps its points.
const PointsKept = 7 * 24 * time.Hour

var validPointName = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)

// ParsePoint reads a point from a message's text, and checks it.
func ParsePoint(text string) (Point, error) {
	var point Point
	if err := json.Unmarshal([]byte(text), &point); err != nil {
		return Point{}, failure.Wrap(NotAPoint, "not JSON", err)
	}
	if !validPointName.MatchString(point.Name) {
		return Point{}, failure.Wrap(NotAPoint, "name "+point.Name, nil)
	}
	if math.IsNaN(point.Value) || math.IsInf(point.Value, 0) {
		return Point{}, failure.Wrap(NotAPoint, "value", nil)
	}
	return point, nil
}

// asPoint checks a text published to a metrics topic, fills in its time when it has none, and
// gives the text to keep, so every reader sees the same time.
func asPoint(text string, now time.Time) (string, error) {
	point, err := ParsePoint(text)
	if err != nil {
		return "", err
	}
	if point.At == 0 {
		point.At = now.UnixMilli()
	}
	written, err := json.Marshal(point)
	if err != nil {
		return "", failure.Wrap(NotAPoint, "unwritable", err)
	}
	return string(written), nil
}
