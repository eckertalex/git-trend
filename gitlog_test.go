package main

import (
	"strings"
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	const us = "\x1f"
	out := "abc123" + us + "Alex Eckert" + us + "alex@example.com" + us + "2024-01-02T10:00:00+01:00\n" +
		"def456" + us + "Sam, Tester" + us + "sam@example.com" + us + "2024-03-04T18:30:00Z\n"

	commits, err := parseCommits(strings.NewReader(out))
	if err != nil {
		t.Fatalf("parseCommits returned error: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}

	want := Commit{
		Hash:        "abc123",
		AuthorName:  "Alex Eckert",
		AuthorEmail: "alex@example.com",
		When:        time.Date(2024, 1, 2, 10, 0, 0, 0, time.FixedZone("", 3600)),
	}
	got := commits[0]
	if got.Hash != want.Hash || got.AuthorName != want.AuthorName || got.AuthorEmail != want.AuthorEmail {
		t.Errorf("commit[0] = %+v, want %+v", got, want)
	}
	if !got.When.Equal(want.When) {
		t.Errorf("commit[0].When = %v, want %v", got.When, want.When)
	}

	// Author name containing a comma must survive intact.
	if commits[1].AuthorName != "Sam, Tester" {
		t.Errorf("commit[1].AuthorName = %q, want %q", commits[1].AuthorName, "Sam, Tester")
	}
}

func TestParseEmpty(t *testing.T) {
	commits, err := parseCommits(strings.NewReader(""))
	if err != nil {
		t.Fatalf("parseCommits(\"\") returned error: %v", err)
	}
	if len(commits) != 0 {
		t.Fatalf("expected 0 commits, got %d", len(commits))
	}
}

func TestParseMalformed(t *testing.T) {
	if _, err := parseCommits(strings.NewReader("only-one-field\n")); err == nil {
		t.Fatal("expected error for malformed line, got nil")
	}
}
