// Copyright (c) 2026 blairtcg
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package velo

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles defines the visual appearance of log entries when using the TextFormatter.
//
// It leverages the lipgloss library to provide rich, customizable terminal styling
// for timestamps, levels, messages, keys, values, and stack traces.
type Styles struct {
	Timestamp lipgloss.Style
	Caller    lipgloss.Style
	Prefix    lipgloss.Style
	Message   lipgloss.Style
	Key       lipgloss.Style
	Value     lipgloss.Style
	Separator lipgloss.Style
	StackFunc lipgloss.Style
	StackFile lipgloss.Style
	Levels    map[Level]lipgloss.Style
	Keys      map[string]lipgloss.Style
	Values    map[string]lipgloss.Style

	// CachedLevelStrings stores the rendered level strings to avoid rendering again on every log.
	// This optimization significantly improves text formatting performance.
	CachedLevelStrings map[Level]string
}

// DefaultStyles initializes and returns the standard styling configuration.
//
// It provides a clean, readable, and color coded default appearance for text logs.
func DefaultStyles() *Styles {
	s := &Styles{
		Timestamp: lipgloss.NewStyle(),
		Caller:    lipgloss.NewStyle().Faint(true),
		Prefix:    lipgloss.NewStyle().Bold(true).Faint(true),
		Message:   lipgloss.NewStyle(),
		Key:       lipgloss.NewStyle().Faint(true),
		Value:     lipgloss.NewStyle(),
		Separator: lipgloss.NewStyle().Faint(true),
		StackFunc: lipgloss.NewStyle().Foreground(lipgloss.Color("252")),
		StackFile: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Levels: map[Level]lipgloss.Style{
			DebugLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(DebugLevel.String())).
				Bold(true).
				MaxWidth(4).
				Foreground(lipgloss.Color("63")),
			InfoLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(InfoLevel.String())).
				Bold(true).
				MaxWidth(4).
				Foreground(lipgloss.Color("86")),
			WarnLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(WarnLevel.String())).
				Bold(true).
				MaxWidth(4).
				Foreground(lipgloss.Color("192")),
			ErrorLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(ErrorLevel.String())).
				Bold(true).
				MaxWidth(4).
				Foreground(lipgloss.Color("204")),
			FatalLevel: lipgloss.NewStyle().
				SetString(strings.ToUpper(FatalLevel.String())).
				Bold(true).
				MaxWidth(4).
				Foreground(lipgloss.Color("134")),
		},
		Keys:   map[string]lipgloss.Style{},
		Values: map[string]lipgloss.Style{},
	}

	s.CachedLevelStrings = make(map[Level]string, len(s.Levels))
	for l, style := range s.Levels {
		s.CachedLevelStrings[l] = style.String()
	}

	return s
}

// SetDefaultStyles overrides the global default styles for the TextFormatter.
//
// You can use this to apply a custom, application wide theme to all text logs.
func SetDefaultStyles(s *Styles) {
	if s == nil {
		return
	}
	// Ensure CachedLevelStrings is populated
	if s.CachedLevelStrings == nil {
		s.CachedLevelStrings = make(map[Level]string, len(s.Levels))
		for l, style := range s.Levels {
			s.CachedLevelStrings[l] = style.String()
		}
	}
	_defaultStyles = s
}
