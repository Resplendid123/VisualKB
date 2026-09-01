package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"learn/internal/domain/conversation"
)

var Default = NewRegistry()

type Registry struct {
	mu     sync.RWMutex
	skills map[string]conversation.Skill
}

func NewRegistry() *Registry {
	return &Registry{skills: make(map[string]conversation.Skill)}
}

func (r *Registry) List() []conversation.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]conversation.Skill, 0, len(r.skills))
	for _, s := range r.skills {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

func (r *Registry) Get(name string) (conversation.Skill, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.skills[name]
	return s, ok
}

func (r *Registry) Register(s conversation.Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.skills[s.Name]; ok {
		return fmt.Errorf("duplicate skill name %s", s.Name)
	}
	r.skills[s.Name] = s
	return nil
}

func (r *Registry) MustRegister(s conversation.Skill) {
	if err := r.Register(s); err != nil {
		panic(err)
	}
}

func (r *Registry) Load(dir string) error {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read skill dir %s: %w", dir, err)
	}
	for _, e := range entries {
		name := e.Name()

		if strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {

			if err := r.loadSkillFromDir(filepath.Join(dir, name)); err != nil {
				return err
			}
			continue
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read skill %s: %w", name, err)
		}
		s, err := parseSkill(name, string(raw))
		if err != nil {
			return fmt.Errorf("parse skill %s: %w", name, err)
		}
		if err := r.Register(s); err != nil {
			return err
		}
	}
	return nil
}

func (r *Registry) loadSkillFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read skill subdir %s: %w", dir, err)
	}
	var cands []string
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		cands = append(cands, e.Name())
	}
	if len(cands) == 0 {
		return nil
	}
	sort.Strings(cands)
	pick := cands[0]
	for _, n := range cands {
		if n == "SKILL.md" {
			pick = n
			break
		}
	}
	raw, err := os.ReadFile(filepath.Join(dir, pick))
	if err != nil {
		return fmt.Errorf("read skill %s: %w", pick, err)
	}

	dirBase := filepath.Base(dir)
	s, err := parseSkill(dirBase, string(raw))
	if err != nil {
		return fmt.Errorf("parse skill %s: %w", dirBase, err)
	}
	return r.Register(s)
}

func parseSkill(filename, raw string) (conversation.Skill, error) {
	var fm skillFrontmatter
	if !strings.HasPrefix(raw, "---") {
		return conversation.Skill{}, fmt.Errorf("missing frontmatter")
	}
	rest := raw[3:]
	rest = strings.TrimLeft(rest, "\r\n")
	end := strings.Index(rest, "\n---")
	if end < 0 {
		return conversation.Skill{}, fmt.Errorf("missing closing frontmatter")
	}
	head := rest[:end]
	body := rest[end+4:]
	body = strings.TrimLeft(body, "\r\n")
	body = strings.TrimRight(body, " \t\r\n")
	if err := yaml.Unmarshal([]byte(head), &fm); err != nil {
		return conversation.Skill{}, fmt.Errorf("yaml frontmatter: %w", err)
	}
	name := strings.TrimSpace(fm.Name)
	if name == "" {
		name = strings.TrimSuffix(filename, ".md")
	}
	if strings.TrimSpace(fm.Description) == "" {
		return conversation.Skill{}, fmt.Errorf("skill %s: description is required", name)
	}
	return conversation.Skill{
		Name:        name,
		Description: strings.TrimSpace(fm.Description),
		Body:        body,
	}, nil
}

type skillFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

var _ conversation.SkillRepo = (*Registry)(nil)
