package filter

type CombineResult struct {
	Domains         []string
	Selectors       []string
	Exceptions      []string
	InjectedCSS     []string
	InjectException []string
}

func Combine(filters []Rule) (m map[string]CombineResult) {
	m = make(map[string]CombineResult)

	for _, f := range filters {
		d := f.JoinedDomains
		out := m[d]

		out.Domains = f.Domains

		if f.CSSSelector != "" && !contains(out.Selectors, f.CSSSelector) {
			if f.isException {
				out.Exceptions = append(out.Exceptions, f.CSSSelector)
			} else {
				out.Selectors = append(out.Selectors, f.CSSSelector)
			}
		}
		if f.InjectedCSS != "" && !contains(out.InjectedCSS, f.InjectedCSS) {
			if f.isException {
				out.InjectException = append(out.InjectedCSS, f.InjectedCSS)
			} else {
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
