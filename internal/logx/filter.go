package logx

import "strings"

type Filter struct {
	terms []filterTerm
}

type filterTerm struct {
	text   string
	negate bool
}

func NewFilter(query string) *Filter {
	query = strings.TrimSpace(query)
	if query == "" {
		return &Filter{}
	}

	parts := strings.Fields(query)
	terms := make([]filterTerm, 0, len(parts))

	for _, part := range parts {
		if part == "" {
			continue
		}
		term := filterTerm{}
		if strings.HasPrefix(part, "!") {
			term.negate = true
			term.text = strings.ToLower(strings.TrimPrefix(part, "!"))
		} else {
			term.text = strings.ToLower(part)
		}
		if term.text != "" {
			terms = append(terms, term)
		}
	}

	return &Filter{terms: terms}
}

func (f *Filter) IsEmpty() bool {
	return len(f.terms) == 0
}

func (f *Filter) Match(text string) bool {
	if f.IsEmpty() {
		return true
	}

	lower := strings.ToLower(text)
	for _, term := range f.terms {
		contains := strings.Contains(lower, term.text)
		if term.negate {
			if contains {
				return false
			}
		} else {
			if !contains {
				return false
			}
		}
	}
	return true
}

func Apply(entries []Entry, query string) []int {
	return ApplyWithLevel(entries, query, nil)
}

func ApplyWithLevel(entries []Entry, query string, levelFilter *Level) []int {
	filter := NewFilter(query)
	result := make([]int, 0, len(entries))

	for i, entry := range entries {
		if entry.Deleted {
			continue
		}
		if levelFilter != nil && entry.Level != *levelFilter {
			continue
		}
		if filter.Match(entry.Raw) {
			result = append(result, i)
		}
	}

	return result
}
