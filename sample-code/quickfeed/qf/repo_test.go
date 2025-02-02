package qf_test

import (
	"testing"

	"github.com/quickfeed/quickfeed/qf"
)

func TestGetTestURL(t *testing.T) {
	want := "https://github.com/dat320-2020/" + qf.TestsRepo
	repo := &qf.Repository{
		HTMLURL: "https://github.com/dat320-2020/meling-labs",
	}
	got := repo.GetTestURL()
	if got != want {
		t.Errorf("GetTestURL() = %s, want %s", got, want)
	}
}

func TestName(t *testing.T) {
	want := "meling-labs"
	repo := &qf.Repository{
		HTMLURL: "https://github.com/dat320-2020/" + want,
	}
	got := repo.Name()
	if got != want {
		t.Errorf("Name() = %s, want %s", got, want)
	}
}

func TestUserName(t *testing.T) {
	want := "meling"
	repo := &qf.Repository{
		HTMLURL: "https://github.com/dat320-2020/" + qf.StudentRepoName(want),
	}
	got := repo.UserName()
	if got != want {
		t.Errorf("UserName() = %s, want %s", got, want)
	}
}
