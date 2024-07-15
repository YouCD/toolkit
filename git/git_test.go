package git

import (
	"testing"
)

func TestGit_PlainClone(t *testing.T) {
	init, err := NewGitInit("http://gitlab.firecloud.wan/sscra/sscra.git", "dev")
	if err != nil {
		t.Error(err)
	}
	err = init.PlainClone("/tmp/test", 1)
	if err != nil {
		t.Error(err)
	}
}
