package conversation

// Loaded into memory once at startup.
type Skill struct {
	Name        string
	Description string
	Body        string
}

type SkillRepo interface {
	List() []Skill
	Get(name string) (Skill, bool)
}
