package theme

import (
	"os/exec"
	"strings"
)

var (
	Search   = "/"
	Active   = "●"
	Inactive = "○"
	SortAsc  = "▲"
	SortDesc = "▼"
	Project  = "◆"
	Session  = "◇"
	Tool     = "⚡"
	Error    = "✗"
	Success  = "✓"
	Arrow    = "▸"
	Clock    = "◷"
	Model    = "◈"
	Branch   = "⑂"
	Tag      = "#"
	Token    = "≋"
	Cost     = "$"
	ID       = "⊞"
	Repo     = "⊙"
	Meta     = "≡"
	Parent   = "↑"
)

func Init(override *bool) {
	var useNerd bool
	if override != nil {
		useNerd = *override
	} else {
		useNerd = detectNerdFont()
	}
	if useNerd {
		initNerdFont()
	}
}

func detectNerdFont() bool {
	out, err := exec.Command("fc-list").Output()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), "nerd font")
}

func initNerdFont() {
	Search = "\ue68f"
	Active = "\uf111"
	Inactive = "\uf10c"
	SortAsc = "\uf0de"
	SortDesc = "\uf0dd"
	Project = "\uf07b"
	Session = "\ue795"
	Tool = "\uf0ad"
	Error = "\uf00d"
	Success = "\uf00c"
	Arrow = "\uf054"
	Clock = "\uf017"
	Model = "\uf2db"
	Branch = "\ue725"
	Tag = "\uf02c"
	Token = "\uf292"
	Cost = "\uf155"
	ID = "\uf2c1"
	Repo = "\uf1d3"
	Meta = "\uf0c9"
	Parent = "\uf062"
}
