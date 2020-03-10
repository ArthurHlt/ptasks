package ptasks

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/mattn/go-shellwords"
)

func ParseCommands(cmdRaw string) ([][]string, error) {
	cmds := make([][]string, 0)
	scanner := bufio.NewScanner(bytes.NewBufferString(cmdRaw))
	for scanner.Scan() {
		txt := strings.TrimSpace(scanner.Text())
		if txt == "" || strings.HasPrefix(txt, "#") {
			continue
		}
		cmdParse, err := shellwords.Parse(txt)
		if err != nil {
			return cmds, err
		}
		cmds = append(cmds, cmdParse)
	}
	return cmds, nil
}
