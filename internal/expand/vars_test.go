package expand

import "testing"

func TestExpandVarsUsesDefault(t *testing.T) {
	defaults := map[string]string{"LOGDIR": "/var/log"}
	t.Setenv("LOGDIR", "")

	got, err := ExpandVars("${LOGDIR}/app", "p1", "roots", defaults)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/var/log/app" {
		t.Errorf("got %q, want %q", got, "/var/log/app")
	}
}

func TestExpandVarsEnvOverridesDefault(t *testing.T) {
	defaults := map[string]string{"LOGDIR": "/var/log"}
	t.Setenv("LOGDIR", "/custom")

	got, err := ExpandVars("${LOGDIR}/app", "p1", "roots", defaults)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/custom/app" {
		t.Errorf("got %q, want %q", got, "/custom/app")
	}
}

func TestExpandVarsPercentForm(t *testing.T) {
	defaults := map[string]string{"LOGDIR": "/var/log"}
	t.Setenv("LOGDIR", "")

	got, err := ExpandVars("%LOGDIR%/app", "p1", "roots", defaults)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/var/log/app" {
		t.Errorf("got %q, want %q", got, "/var/log/app")
	}
}

func TestExpandVarsUnknownVariable(t *testing.T) {
	defaults := map[string]string{}
	_, err := ExpandVars("${FOO}", "p1", "roots", defaults)
	if err == nil {
		t.Fatal("expected error for unknown variable")
	}
}

func TestExpandVarsNoValueNoDefault(t *testing.T) {
	defaults := map[string]string{"FOO": ""}
	t.Setenv("FOO", "")

	_, err := ExpandVars("${FOO}", "p1", "roots", defaults)
	if err == nil {
		t.Fatal("expected error for variable without value and default")
	}
}

func TestExpandVarsNoRecursiveSubstitution(t *testing.T) {
	defaults := map[string]string{
		"A": "${B}",
		"B": "val",
	}
	t.Setenv("A", "")
	t.Setenv("B", "")

	got, err := ExpandVars("${A}", "p1", "roots", defaults)
	if err != nil {
		t.Fatal(err)
	}
	if got != "${B}" {
		t.Errorf("got %q, want %q", got, "${B}")
	}
}
