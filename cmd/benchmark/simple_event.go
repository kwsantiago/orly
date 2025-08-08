package main

import (
	"fmt"

	"lukechampine.com/frand"
	"orly.dev/pkg/encoders/event"
	"orly.dev/pkg/encoders/kind"
	"orly.dev/pkg/encoders/tags"
	"orly.dev/pkg/encoders/timestamp"
	"orly.dev/pkg/utils/chk"
)

func generateSimpleEvent(signer *testSigner, contentSize int) *event.E {
	content := generateContent(contentSize)
	
	ev := &event.E{
		Kind:      kind.TextNote,
		Tags:      tags.New(),
		Content:   []byte(content),
		CreatedAt: timestamp.Now(),
		Pubkey:    signer.Pub(),
	}
	
	if err := ev.Sign(signer); chk.E(err) {
		panic(fmt.Sprintf("failed to sign event: %v", err))
	}
	
	return ev
}

func generateContent(size int) string {
	words := []string{
		"the", "be", "to", "of", "and", "a", "in", "that", "have", "I",
		"it", "for", "not", "on", "with", "he", "as", "you", "do", "at",
		"this", "but", "his", "by", "from", "they", "we", "say", "her", "she",
		"or", "an", "will", "my", "one", "all", "would", "there", "their", "what",
		"so", "up", "out", "if", "about", "who", "get", "which", "go", "me",
		"when", "make", "can", "like", "time", "no", "just", "him", "know", "take",
		"people", "into", "year", "your", "good", "some", "could", "them", "see", "other",
		"than", "then", "now", "look", "only", "come", "its", "over", "think", "also",
		"back", "after", "use", "two", "how", "our", "work", "first", "well", "way",
		"even", "new", "want", "because", "any", "these", "give", "day", "most", "us",
	}
	
	result := ""
	for len(result) < size {
		if len(result) > 0 {
			result += " "
		}
		result += words[frand.Intn(len(words))]
	}
	
	if len(result) > size {
		result = result[:size]
	}
	
	return result
}