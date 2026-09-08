package app

import (
	"context"
	"errors"
	"example.com/greeter/internal/platform/fault"
	"testing"
)

func TestGreeting(t *testing.T) {
	got, err := NewGreetPerson(GreetPersonDeps{}).Execute(context.Background(), GreetPersonInput{Name: "Ben"})
	if err != nil || got.Message != "Hello, Ben!" {
		t.Fatalf("result=%+v error=%v", got, err)
	}
}
func TestRequiredName(t *testing.T) {
	_, err := NewGreetPerson(GreetPersonDeps{}).Execute(context.Background(), GreetPersonInput{Name: "  "})
	if !fault.IsKind(err, fault.Invalid) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}
func TestCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewGreetPerson(GreetPersonDeps{}).Execute(ctx, GreetPersonInput{Name: "Ben"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}
