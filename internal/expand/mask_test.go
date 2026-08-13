package expand

import "testing"

func TestValidateMask(t *testing.T) {
	if err := ValidateMask("*.log"); err != nil {
		t.Errorf("valid mask rejected: %v", err)
	}
	if err := ValidateMask("["); err == nil {
		t.Error("invalid mask accepted")
	}
}

func TestMatch(t *testing.T) {
	ok, err := Match("*.log", "app.log")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Error("expected match for *.log vs app.log")
	}

	ok, err = Match("*.log", "app.txt")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Error("expected no match for *.log vs app.txt")
	}
}
