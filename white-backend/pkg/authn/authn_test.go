package authn

import (
	"testing"
	"time"
)

func TestIssueVerify(t *testing.T) {
	priv, err := KeyPair("", "dev-secret")
	if err != nil {
		t.Fatal(err)
	}
	pub, _ := PublicKey("", "", "dev-secret")
	iss := NewIssuer(priv, 15*time.Minute)
	now := time.Now()
	tok, exp, err := iss.Issue(Principal{UserID: "u1", Role: RoleAdmin, Email: "a@b.c"}, now)
	if err != nil || exp.Sub(now) != 15*time.Minute {
		t.Fatal(err, exp)
	}
	p, err := NewVerifier(pub).Verify(tok)
	if err != nil || p.UserID != "u1" || !p.IsAdmin() || p.Email != "a@b.c" {
		t.Fatalf("%+v %v", p, err)
	}
	other, _ := PublicKey("", "", "other-secret")
	if _, err := NewVerifier(other).Verify(tok); err == nil {
		t.Fatal("verified with the wrong key")
	}
	expired, _, _ := NewIssuer(priv, -time.Hour).Issue(Principal{UserID: "u1"}, now)
	if _, err := NewVerifier(pub).Verify(expired); err == nil {
		t.Fatal("expired token accepted")
	}
}

func TestBearer(t *testing.T) {
	if v, ok := BearerToken("Bearer abc"); !ok || v != "abc" {
		t.Fatal(v, ok)
	}
	if _, ok := BearerToken("Basic abc"); ok {
		t.Fatal("basic accepted")
	}
}
