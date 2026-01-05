package logx

import (
	"reflect"
	"testing"
)

func makeEntries(raws ...string) []Entry {
	entries := make([]Entry, len(raws))
	for i, raw := range raws {
		entries[i] = Entry{Index: i, Raw: raw}
	}
	return entries
}

func TestFilterEmpty(t *testing.T) {
	entries := makeEntries("foo", "bar", "baz")
	result := Apply(entries, "")
	if len(result) != 3 {
		t.Errorf("Empty filter should return all entries, got %d", len(result))
	}
}

func TestFilterSimple(t *testing.T) {
	entries := makeEntries("error: something", "info: all good", "error: another")
	result := Apply(entries, "error")
	expected := []int{0, 2}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestFilterCaseInsensitive(t *testing.T) {
	entries := makeEntries("ERROR: uppercase", "error: lowercase", "ErRoR: mixed")
	result := Apply(entries, "error")
	if len(result) != 3 {
		t.Errorf("Case insensitive filter should match all, got %d", len(result))
	}
}

func TestFilterNegation(t *testing.T) {
	entries := makeEntries("error: bad", "info: good", "debug: trace")
	result := Apply(entries, "!error")
	expected := []int{1, 2}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestFilterAND(t *testing.T) {
	entries := makeEntries(
		"error timeout in database",
		"error network failure",
		"info timeout handled",
	)
	result := Apply(entries, "error timeout")
	expected := []int{0}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestFilterMixed(t *testing.T) {
	entries := makeEntries(
		"error timeout in database",
		"error network failure",
		"info timeout handled",
	)
	result := Apply(entries, "timeout !error")
	expected := []int{2}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %v, want %v", result, expected)
	}
}

func TestFilterDeleted(t *testing.T) {
	entries := makeEntries("foo", "bar", "baz")
	entries[1].Deleted = true
	result := Apply(entries, "")
	expected := []int{0, 2}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deleted entries should be excluded, got %v", result)
	}
}

func TestFilterMatch(t *testing.T) {
	f := NewFilter("goku !frieza")

	if !f.Match("goku is training") {
		t.Error("Should match 'goku is training'")
	}
	if f.Match("goku vs frieza") {
		t.Error("Should not match 'goku vs frieza'")
	}
	if f.Match("vegeta is here") {
		t.Error("Should not match 'vegeta is here'")
	}
}
