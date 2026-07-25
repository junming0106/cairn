// Package skills 找出目前專案可用的 AI 技能（Claude Code skills）。
package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 技能的來源。
const (
	ScopeUser    = "全域"
	ScopeProject = "專案"
	ScopePlugin  = "插件"
)

// Skill 是一個技能。
type Skill struct {
	Name        string
	Description string
	Scope       string
	Origin      string // 插件技能的插件名稱
	Path        string
	Body        string // SKILL.md 去掉 frontmatter 後的正文
}

// Load 掃出專案可用的技能：使用者全域的、專案自己的，以及有啟用的插件帶來的。
func Load(projectRoot string) []Skill {
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	var out []Skill
	if home != "" {
		out = append(out, scanDir(filepath.Join(home, ".claude", "skills"), ScopeUser, "")...)
	}
	if projectRoot != "" {
		out = append(out, scanDir(filepath.Join(projectRoot, ".claude", "skills"), ScopeProject, "")...)
	}
	out = append(out, pluginSkills(home, projectRoot)...)

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Scope != out[j].Scope {
			return scopeRank(out[i].Scope) < scopeRank(out[j].Scope)
		}
		return out[i].Name < out[j].Name
	})
	return out
}

func scopeRank(s string) int {
	switch s {
	case ScopeProject:
		return 0
	case ScopeUser:
		return 1
	default:
		return 2
	}
}

// scanDir 讀取 <dir>/<技能名>/SKILL.md。
func scanDir(dir, scope, origin string) []Skill {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Skill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		p := filepath.Join(dir, e.Name(), "SKILL.md")
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		s := parse(string(b))
		if s.Name == "" {
			s.Name = e.Name()
		}
		s.Scope, s.Origin, s.Path = scope, origin, p
		out = append(out, s)
	}
	return out
}

// pluginSkills 只列出設定檔中有啟用的插件所帶的技能。
func pluginSkills(home, projectRoot string) []Skill {
	enabled := enabledPlugins(home, projectRoot)
	if len(enabled) == 0 || home == "" {
		return nil
	}
	root := filepath.Join(home, ".claude", "plugins", "marketplaces")
	markets, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []Skill
	for _, mk := range markets {
		if !mk.IsDir() {
			continue
		}
		plugins, err := os.ReadDir(filepath.Join(root, mk.Name(), "plugins"))
		if err != nil {
			continue
		}
		for _, pl := range plugins {
			if !pl.IsDir() || !enabled[pl.Name()+"@"+mk.Name()] {
				continue
			}
			out = append(out, scanDir(
				filepath.Join(root, mk.Name(), "plugins", pl.Name(), "skills"),
				ScopePlugin, pl.Name())...)
		}
	}
	return out
}

// enabledPlugins 合併使用者與專案設定裡 enabledPlugins 為 true 的項目。
func enabledPlugins(home, projectRoot string) map[string]bool {
	out := map[string]bool{}
	var files []string
	if home != "" {
		files = append(files, filepath.Join(home, ".claude", "settings.json"))
	}
	if projectRoot != "" {
		files = append(files,
			filepath.Join(projectRoot, ".claude", "settings.json"),
			filepath.Join(projectRoot, ".claude", "settings.local.json"))
	}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		var cfg struct {
			EnabledPlugins map[string]any `json:"enabledPlugins"`
		}
		if json.Unmarshal(b, &cfg) != nil {
			continue
		}
		for id, v := range cfg.EnabledPlugins {
			if on, ok := v.(bool); ok {
				out[id] = on
			} else {
				out[id] = true // 版本限制寫法也算啟用
			}
		}
	}
	return out
}

// parse 取出 SKILL.md 的 frontmatter 與正文。
func parse(src string) Skill {
	var s Skill
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		s.Body = strings.TrimSpace(src)
		return s
	}
	key := ""
	i := 1
	for ; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "---" {
			i++
			break
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			// 續行：接到上一個欄位後面
			if key != "" {
				val := strings.TrimSpace(line)
				switch key {
				case "name":
					s.Name += " " + val
				case "description":
					s.Description += " " + val
				}
			}
			continue
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(k)
		val := strings.Trim(strings.TrimSpace(v), `"'`)
		switch key {
		case "name":
			s.Name = val
		case "description":
			s.Description = val
		}
	}
	s.Body = strings.TrimSpace(strings.Join(lines[i:], "\n"))
	return s
}
