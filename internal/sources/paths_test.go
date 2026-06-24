package sources

import (
	"path/filepath"
	"testing"
)

func TestPathsProjectsRootEnv(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "/custom/cfg")
	got, err := ProjectsRoot()
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join("/custom/cfg", "projects") {
		t.Fatalf("ProjectsRoot = %q", got)
	}
}

func TestPathsDecodeProjectDir(t *testing.T) {
	cases := map[string]string{
		"-home-user-foo": "/home/user/foo",
		"":               "",
	}
	for in, want := range cases {
		if got := DecodeProjectDir(in); got != want {
			t.Fatalf("DecodeProjectDir(%q) = %q, want %q", in, got, want)
		}
	}
}
