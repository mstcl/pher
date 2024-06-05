package convert

import (
	"fmt"
	"slices"
	"testing"
)

func TestHref(t *testing.T) {
	tests := []struct {
		f    string
		inDir string
		want string
	}{
		{"/x/y/z/a/b/c/d.md", "/x/y/z", "a/b/c/d"},
		{"/x/y/z/d.md", "/x/y/z", "d"},
		{"/a/b/c/d.md", "/", "a/b/c/d"},
		{"/d.md", "/", "d"},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%s,%s", tt.f, tt.want)
		t.Run(testname, func(t *testing.T) {
			ans := Href(tt.f, tt.inDir, false)
			if ans != tt.want {
				t.Errorf("got %s, want %s", ans, tt.want)
			}
		})
	}
}

func TestNavCrumbs(t *testing.T) {
	tests := []struct {
		f          string
		inDir      string
		crumbs     []string
		crumbLinks []string
	}{
		{
			"/x/y/z/file.md",
			"/x/y/z",
			[]string{},
			[]string{},
		},
		{
			"/a/b/c/file.md",
			"/",
			[]string{"a", "b", "c"},
			[]string{"a/index.html", "a/b/index.html", "a/b/c/index.html"},
		},
		{
			"/x/y/z/a/b/c/file.md",
			"/x/y/z",
			[]string{"a", "b", "c"},
			[]string{"a/index.html", "a/b/index.html", "a/b/c/index.html"},
		},
	}

	for _, tt := range tests {
		testname := fmt.Sprint(tt.f)
		t.Run(testname, func(t *testing.T) {
			crumbs, crumbLinks := NavCrumbs(tt.f, tt.inDir, true)
			if !slices.Equal(crumbs, tt.crumbs) {
				t.Errorf("got %s, want %s", crumbs, tt.crumbs)
			}
			if !slices.Equal(crumbLinks, tt.crumbLinks) {
				t.Errorf("got %s, want %s", crumbLinks, tt.crumbLinks)
			}
		})
	}
}

func TestTitle(t *testing.T) {
	tests := []struct {
		mt, fn string
		want   string
	}{
		{"", "a", "a"},
		{"a", "b", "a"},
		{"a", "", "a"},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%s,%s,%s", tt.mt, tt.fn, tt.want)
		t.Run(testname, func(t *testing.T) {
			ans := Title(tt.mt, tt.fn)
			if ans != tt.want {
				t.Errorf("got %s, want %s", ans, tt.want)
			}
		})
	}
}

func TestFileBase(t *testing.T) {
	tests := []struct {
		f    string
		want string
	}{
		{"a/b/c/d.md", "d"},
		{"/a/b/c/d.md", "d"},
		{"/a/b/c/d", "d"},
		{"/a/b/c/d/", "d"},
	}

	for _, tt := range tests {
		testname := fmt.Sprintf("%s,%s", tt.f, tt.want)
		t.Run(testname, func(t *testing.T) {
			ans := FileBase(tt.f)
			if ans != tt.want {
				t.Errorf("got %s, want %s", ans, tt.want)
			}
		})
	}
}
