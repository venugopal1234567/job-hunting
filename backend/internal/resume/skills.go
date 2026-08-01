package resume

import "strings"

// skillTaxonomy is the master list of recognized technology skills
var skillTaxonomy = []string{
	// Languages
	"Go", "Golang", "Python", "TypeScript", "JavaScript", "Rust", "Java",
	"C++", "C#", "Ruby", "PHP", "Swift", "Kotlin", "Scala", "Elixir",
	// Databases
	"PostgreSQL", "MySQL", "SQLite", "MongoDB", "Redis", "Elasticsearch",
	"Cassandra", "DynamoDB", "CockroachDB", "ClickHouse",
	// DevOps & Cloud
	"Docker", "Kubernetes", "Helm", "Terraform", "Ansible",
	"AWS", "GCP", "Azure", "CI/CD", "GitHub Actions", "GitLab CI",
	"Jenkins", "ArgoCD", "Prometheus", "Grafana", "Datadog",
	// Frameworks & Tools
	"Gin", "Echo", "Fiber", "FastAPI", "Django", "Flask",
	"React", "Vue", "Angular", "Next.js", "Svelte",
	"Node.js", "Express", "gRPC", "GraphQL", "REST API",
	"Kafka", "RabbitMQ", "NATS", "WebSocket",
	// Architecture
	"Microservices", "Event-Driven", "CQRS", "DDD",
	// Tools
	"Git", "Linux", "Nginx", "HAProxy", "Istio",
	"SQL", "Bash", "Shell Scripting",
}

// ExtractSkills scans resume text and returns all recognized skills
func ExtractSkills(text string) []string {
	textLower := strings.ToLower(text)
	found := map[string]bool{}
	result := []string{}

	for _, skill := range skillTaxonomy {
		// Match skill as whole word (case-insensitive)
		skillLower := strings.ToLower(skill)
		if containsWord(textLower, skillLower) {
			if !found[skill] {
				found[skill] = true
				result = append(result, skill)
			}
		}
	}

	return result
}

// containsWord checks if a text contains a skill as a separate word/token
func containsWord(text, word string) bool {
	// Simple word boundary check: preceded/followed by non-alphanumeric
	idx := 0
	for {
		pos := strings.Index(text[idx:], word)
		if pos < 0 {
			return false
		}
		pos += idx
		start := pos
		end := pos + len(word)

		// Check boundaries
		prevOK := start == 0 || !isAlphaNum(rune(text[start-1]))
		nextOK := end >= len(text) || !isAlphaNum(rune(text[end]))

		if prevOK && nextOK {
			return true
		}
		idx = pos + 1
		if idx >= len(text) {
			return false
		}
	}
}

func isAlphaNum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
