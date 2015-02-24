package ansi

import (
	"fmt"
	"testing"
)

func TestStripAnsiControl(t *testing.T) {

	// Up cursor
	stripped := string(StripAnsiControl([]byte("\033[4A")))
	if stripped != "" {
		fmt.Printf("%#v", stripped)
		t.Fatalf("Expected ''. Got %s", stripped)
	}

	// Down cursor
	stripped = string(StripAnsiControl([]byte("\033[4B")))
	if stripped != "" {
		fmt.Printf("%#v", stripped)
		t.Fatalf("Expected ''. Got %s", stripped)
	}

	// Forward cursor
	stripped = string(StripAnsiControl([]byte("\033[4C")))
	if stripped != "" {
		fmt.Printf("%#v", stripped)
		t.Fatalf("Expected ''. Got %s", stripped)
	}

	// Backward cursor
	stripped = string(StripAnsiControl([]byte("\033[4D")))
	if stripped != "" {
		fmt.Printf("%#v", stripped)
		t.Fatalf("Expected ''. Got %s", stripped)
	}

	// Cursor position
	stripped = string(StripAnsiControl([]byte("\033[4;4H")))
	if stripped != "" {
		fmt.Printf("%#v", stripped)
		t.Fatalf("Expected ''. Got %s", stripped)
	}
}

func TestDontStripColor(t *testing.T) {
	// Up cursor
	stripped := string(StripAnsiControl([]byte("\033[4;30m")))
	if stripped != "\033[4;30m" {
		fmt.Printf("%#v", stripped)
		t.Fatalf("Expected ''. Got %s", stripped)
	}
}

func TestNonAnsi(t *testing.T) {
	// Up cursor
	stripped := string(StripAnsiControl([]byte("test")))
	if stripped != "test" {
		fmt.Printf("%#v", stripped)
		t.Fatalf("Expected ''. Got %s", stripped)
	}
}

func TestMixedColor(t *testing.T) {
	// Up cursor
	stripped := string(StripAnsiControl([]byte("\033[0;34mtest\033[0m")))
	if stripped != "\033[0;34mtest\033[0m" {
		fmt.Printf("%#v", stripped)
		t.Fatalf("Expected '\033[0;34mtest\033[0m'. Got %s", stripped)
	}
}
