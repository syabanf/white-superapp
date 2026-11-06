package mail

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestHTMLBodyEscapesAndLinks(t *testing.T) {
	got := htmlBody(Message{Text: "Halo <b>\nhttps://mixist.id/share/a?x=1\n\nBaris dua"})
	for _, want := range []string{"Halo &lt;b&gt;<br>", `<a href="https://mixist.id/share/a?x=1">`, "<p>Baris dua</p>"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %s", want, got)
		}
	}
}

func TestRecipientsDropsBlank(t *testing.T) {
	if got := recipients([]string{" a@b.id ", "", "nope"}); len(got) != 1 || got[0] != "a@b.id" {
		t.Fatal(got)
	}
}

func TestResendWithoutKeyOnlyLogs(t *testing.T) {
	r := New(func(context.Context) (string, string, error) { return "", "", nil }, nil)
	if err := r.Send(context.Background(), Message{To: []string{"a@b.id"}, Subject: "x"}); err != nil {
		t.Fatal(err)
	}
	failing := New(func(context.Context) (string, string, error) { return "", "", errors.New("down") }, nil)
	if err := failing.Send(context.Background(), Message{To: []string{"a@b.id"}}); err == nil {
		t.Fatal("credential failure must surface")
	}
}
