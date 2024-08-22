package filter

import (
	"strings"
)

type CombineResult struct {
	Domains            []string
	Selectors          []string
	Exceptions         []string
	InjectedCSS        []string
	InjectionException []string
}

func Combine(filters []Rule) (m map[string]CombineResult) {
	m = make(map[string]CombineResult)

	for _, f := range filters {
		d := strings.Join(f.Domains, ",")
		out := m[d]

		out.Domains = f.Domains

		if f.CSSSelector != "" {
			if f.isException && !contains(out.Exceptions, f.CSSSelector) {
				out.Exceptions = append(out.Exceptions, f.CSSSelector)
			} else if !contains(out.Selectors, f.CSSSelector) {
				out.Selectors = append(out.Selectors, f.CSSSelector)
			}
		}
		if f.InjectedCSS != "" {
			if f.isException && !contains(out.InjectionException, f.InjectedCSS) {
				out.InjectionException = append(out.InjectionException, f.InjectedCSS)
			} else if !contains(out.InjectedCSS, f.InjectedCSS) {
				out.InjectedCSS = append(out.InjectedCSS, f.InjectedCSS)
			}
		}

		m[d] = out
	}

	return m
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
