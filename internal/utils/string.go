package utils

import (
	"net/url"
	"strings"
	"unicode"
)

func TruncateTitle(title string, max int) string {
	if len(title) <= max {
		return title
	}
	return title[:max] + "..."
}

func ExtractDomain(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}

	host := u.Hostname()
	// Pisah jadi bagian-bagian
	parts := strings.Split(host, ".")

	// Ambil bagian tengah (contoh: kontan dari investasi.kontan.co.id)
	// Rule umum: ambil bagian ke-1 dari belakang jika .co.id atau .com
	// Contoh: bloombergtechnoz.com → bloombergtechnoz
	if len(parts) >= 3 && parts[len(parts)-2] == "co" {
		return parts[len(parts)-3] // kontan dari investasi.kontan.co.id
	} else if len(parts) >= 2 {
		return parts[len(parts)-2] // bloombergtechnoz dari bloombergtechnoz.com
	}

	// Fallback
	return host

}

func SummarizeIssues(issues []string, max int) string {
	if len(issues) > max {
		issues = issues[:max]
	}
	return strings.Join(issues, ", ")
}

func CapitalizeSentence(input string) string {
	input = strings.TrimSpace(input)
	if input == "" {
		return ""
	}

	runes := []rune(input)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func EscapeMarkdownV2(text string) string {
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		".", "\\.",
		"!", "\\!",
	)
	return replacer.Replace(text)
}
