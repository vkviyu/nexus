package domain

// Settings represents user preferences and application configuration.
type Settings struct {
	Theme    string `json:"theme"`    // dark, light
	Language string `json:"language"` // zh, en
}

// NewDefaultSettings creates Settings with default values.
func NewDefaultSettings() *Settings {
	return &Settings{
		Theme:    "dark",
		Language: "zh",
	}
}

// Validate ensures settings have valid values, applying defaults if needed.
func (s *Settings) Validate() {
	if s.Theme == "" {
		s.Theme = "dark"
	}
	if s.Language == "" {
		s.Language = "zh"
	}
}