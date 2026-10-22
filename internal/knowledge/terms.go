package knowledge

import "strings"

// Terminology supplies technical aliases, not generated facts or executable
// instructions. It helps the local model resolve Arabic domain ambiguity.
func Terminology(query string) string {
	normalized := Normalize(query)
	terms := []struct {
		aliases []string
		english string
	}{
		{[]string{"اقل امتياز", "الحد الادني من الامتيازات", "اقل صلاحيات"}, "Principle of Least Privilege (computer security access control)"},
		{[]string{"حقن sql", "حقن الاستعلامات"}, "SQL Injection"},
		{[]string{"تزوير الطلبات"}, "Cross-Site Request Forgery (CSRF)"},
		{[]string{"البرمجه عبر المواقع", "البرمجة عبر المواقع"}, "Cross-Site Scripting (XSS)"},
	}
	var hints []string
	for _, t := range terms {
		for _, alias := range t.aliases {
			if strings.Contains(normalized, Normalize(alias)) {
				hints = append(hints, alias+" = "+t.english)
				break
			}
		}
	}
	if len(hints) == 0 {
		return ""
	}
	return "Technical terminology aliases (not execution instructions):\n" + strings.Join(hints, "\n") + "\n"
}
