package version

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionPrint(t *testing.T) {
	Version()
}

func TestVersionGen(t *testing.T) {
	version := os.Getenv("VERSION")
	if version == "" {
		version = "v0.0.0"
	}
	branch, err := cmdRun("git branch --show-current")
	assert.Nil(t, err)
	commit, err := cmdRun("git rev-parse HEAD")
	assert.Nil(t, err)

	versionstr := version + "-" + branch + "-" + commit
	err = os.WriteFile("VERSION", []byte(versionstr), 0644)
	assert.Nil(t, err)
}

func cmdRun(cmdstr string) (string, error) {
	var buf bytes.Buffer
	cmds := strings.Split(cmdstr, " ")
	if len(cmds) == 1 {
		cmd := exec.Command(cmds[0])
		cmd.Stdout = &buf
		err := cmd.Run()
		if err != nil {
			return "", err
		}
	} else {
		cmd := exec.Command(cmds[0], cmds[1:]...)
		cmd.Stdout = &buf
		err := cmd.Run()
		if err != nil {
			return "", err
		}
	}

	return strings.TrimSpace(buf.String()), nil
}
