// Package i18n holds every string the offline tool shows to a person.
//
// The catalogue is one map per language, so adding a language means adding one
// file and touching no template. English is the default and the fallback: a key
// missing from a translation falls back to English rather than rendering blank,
// because a half-translated page is still usable and an empty one is not.
//
// Messages may contain HTML (the runbook is a formatted document), so T returns
// template.HTML. That is safe because the catalogue is compiled-in source, but
// it means arguments are escaped before substitution: names, bank details and
// other user input reach these messages through the format arguments.
package i18n

import (
	"fmt"
	"html/template"
	"sort"
)

// Lang is a language tag as used in URLs and the runbook form.
type Lang string

const (
	EN Lang = "en"
	SK Lang = "sk"
	CS Lang = "cs"
)

// Default is used when nothing else is asked for, and as the fallback for any
// key a translation is missing.
const Default = EN

var catalogs = map[Lang]map[string]string{
	EN: enMessages,
	SK: skMessages,
	CS: csMessages,
}

// Languages lists the available languages in display order.
func Languages() []Lang { return []Lang{EN, SK, CS} }

// Name is the language's own name, for a picker.
func Name(l Lang) string {
	switch l {
	case SK:
		return "Slovensky"
	case CS:
		return "Česky"
	default:
		return "English"
	}
}

// Parse turns a query parameter or form value into a language, falling back to
// the default for anything unknown.
func Parse(s string) Lang {
	l := Lang(s)
	if _, ok := catalogs[l]; ok {
		return l
	}
	return Default
}

// T looks up a message and substitutes any arguments, escaping them first.
func T(l Lang, key string, args ...any) template.HTML {
	f, ok := catalogs[l][key]
	if !ok || f == "" {
		f, ok = catalogs[Default][key]
		if !ok {
			// A missing key is a bug in this package, not something to hide
			// behind an empty page.
			return template.HTML("[[" + key + "]]")
		}
	}
	if len(args) == 0 {
		return template.HTML(f)
	}
	esc := make([]any, len(args))
	for i, a := range args {
		esc[i] = template.HTMLEscapeString(fmt.Sprint(a))
	}
	return template.HTML(fmt.Sprintf(f, esc...))
}

// S is T without HTML, for log lines and plain-text errors.
func S(l Lang, key string, args ...any) string {
	f, ok := catalogs[l][key]
	if !ok || f == "" {
		if f, ok = catalogs[Default][key]; !ok {
			return "[[" + key + "]]"
		}
	}
	if len(args) == 0 {
		return f
	}
	return fmt.Sprintf(f, args...)
}

// Keys lists every key in a catalogue, sorted. Used by the drift test.
func Keys(l Lang) []string {
	out := make([]string, 0, len(catalogs[l]))
	for k := range catalogs[l] {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
