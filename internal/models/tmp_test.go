package models

import (
	"encoding/json"
	"testing"
)

func TestMythicRatingTolerantTemp(t *testing.T) {
	obj := CharacterData{Challenge: json.RawMessage(`{"current_score":1687.35}`)}
	if got := obj.MythicRating(); got != 1687.35 {
		t.Fatalf("object rating=%v", got)
	}

	arr := CharacterData{Challenge: json.RawMessage(`[]`)}
	if got := arr.MythicRating(); got != 0 {
		t.Fatalf("array should give 0, got %v", got)
	}

	null := CharacterData{Challenge: json.RawMessage(`null`)}
	if got := null.MythicRating(); got != 0 {
		t.Fatalf("null should give 0, got %v", got)
	}

	empty := CharacterData{}
	if got := empty.MythicRating(); got != 0 {
		t.Fatalf("empty should give 0, got %v", got)
	}
}