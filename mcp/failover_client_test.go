package mcp

import (
	"fmt"
	"testing"
	"time"
)

type stubAIClient struct {
	resp string
	err  error

	callsMessages int
	callsRequest  int
}

func (s *stubAIClient) SetAPIKey(apiKey string, customURL string, customModel string) {}
func (s *stubAIClient) SetTimeout(timeout time.Duration)                              {}
func (s *stubAIClient) CallWithMessages(systemPrompt, userPrompt string) (string, error) {
	s.callsMessages++
	if s.err != nil {
		return "", s.err
	}
	return s.resp, nil
}
func (s *stubAIClient) CallWithRequest(req *Request) (string, error) {
	s.callsRequest++
	if s.err != nil {
		return "", s.err
	}
	return s.resp, nil
}

func TestFailoverClient_UsesBackupOn500(t *testing.T) {
	primary := &stubAIClient{err: fmt.Errorf("API returned error (status 500): boom")}
	backup := &stubAIClient{resp: "ok"}
	fc := NewFailoverClient(primary, backup)
	fc.logger = NewNoopLogger()
	fc.cooldown = time.Hour

	got, err := fc.CallWithMessages("sys", "user")
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	if got != "ok" {
		t.Fatalf("expected ok, got: %q", got)
	}
	if primary.callsMessages != 1 || backup.callsMessages != 1 {
		t.Fatalf("expected calls primary=1 backup=1, got primary=%d backup=%d", primary.callsMessages, backup.callsMessages)
	}
}

func TestFailoverClient_DoesNotFailoverOn401(t *testing.T) {
	primary := &stubAIClient{err: fmt.Errorf("API returned error (status 401): unauthorized")}
	backup := &stubAIClient{resp: "ok"}
	fc := NewFailoverClient(primary, backup)
	fc.logger = NewNoopLogger()

	_, err := fc.CallWithMessages("sys", "user")
	if err == nil {
		t.Fatalf("expected error")
	}
	if backup.callsMessages != 0 {
		t.Fatalf("expected backup not called, got: %d", backup.callsMessages)
	}
}

func TestFailoverClient_DisablesTransientClient(t *testing.T) {
	primary := &stubAIClient{err: fmt.Errorf("API returned error (status 500): boom")}
	backup := &stubAIClient{resp: "ok"}
	fc := NewFailoverClient(primary, backup)
	fc.logger = NewNoopLogger()
	fc.cooldown = time.Hour

	// First call disables primary and succeeds via backup.
	_, err := fc.CallWithMessages("sys", "user")
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	if primary.callsMessages != 1 || backup.callsMessages != 1 {
		t.Fatalf("expected calls primary=1 backup=1, got primary=%d backup=%d", primary.callsMessages, backup.callsMessages)
	}

	// Primary would succeed now, but it's still in cooldown, so backup should be used.
	primary.err = nil
	primary.resp = "primary"
	backup.resp = "backup"

	got, err := fc.CallWithMessages("sys", "user")
	if err != nil {
		t.Fatalf("expected success, got err: %v", err)
	}
	if got != "backup" {
		t.Fatalf("expected backup, got: %q", got)
	}
	if primary.callsMessages != 1 {
		t.Fatalf("expected primary still not called due to cooldown, got: %d", primary.callsMessages)
	}
	if backup.callsMessages != 2 {
		t.Fatalf("expected backup called twice, got: %d", backup.callsMessages)
	}
}

