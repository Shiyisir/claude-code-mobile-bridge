package security

import (
	"os"
	"regexp"
	"strings"
)

// allowedEnv is the whitelist of env vars safe to log.
var allowedEnv = map[string]bool{
	"PATH":                        true,
	"ComSpec":                     true,
	"SHELL":                       true,
	"TERM":                        true,
	"WT_SESSION":                  true,
	"VSCODE_PID":                  true,
	"ELECTRON_RUN_AS_NODE":        true,
	"CLAUDE_PROXY_REAL_BIN":       true,
	"USERPROFILE":                 true,
	"HOMEDRIVE":                   true,
	"HOMEPATH":                    true,
	"APPDATA":                     true,
	"LOCALAPPDATA":                true,
	"TEMP":                        true,
	"TMP":                         true,
	"OS":                          true,
	"PROCESSOR_ARCHITECTURE":      true,
	"SYSTEMROOT":                  true,
	"USERNAME":                    true,
	"COMPUTERNAME":                true,
	"PATHEXT":                     true,
	"NO_PROXY":                    true,
	"HTTP_PROXY":                  true,
	"HTTPS_PROXY":                 true,
}

// blockedSuffixes are case-insensitive suffixes of env vars that must not be logged.
var blockedSuffixes = []string{
	"_TOKEN",
	"_KEY",
	"_SECRET",
	"PASSWORD",
	"PWD",
	"CREDENTIAL",
	"COOKIE",
	"AUTHORIZATION",
}

// IsEnvAllowed reports whether an environment variable name is safe to record.
func IsEnvAllowed(name string) bool {
	if allowedEnv[name] {
		return true
	}
	upper := strings.ToUpper(name)
	for _, s := range blockedSuffixes {
		if strings.HasSuffix(upper, strings.ToUpper(s)) {
			return false
		}
	}
	// Conservative: unknown vars are not logged
	return false
}

// apiKeyPattern matches Anthropic-style API keys (sk-...), OpenAI keys (sk-...),
// and common Bearer/JWT tokens in JSON strings.
var redactPatterns = []*regexp.Regexp{
	// Anthropic/OpenAI API keys: "sk-..." in JSON values
	regexp.MustCompile(`"(?:ANTHROPIC_API_KEY|ANTHROPIC_AUTH_TOKEN|OPENAI_API_KEY|GITHUB_TOKEN|API_KEY|AUTH_TOKEN|BEARER_TOKEN)"\s*:\s*"sk-[^"]+"`),
	// Generic credential keys with any value
	regexp.MustCompile(`"([^"]*(?:KEY|TOKEN|SECRET|PASSWORD|AUTHORIZATION)[^"]*)"\s*:\s*"[^"]*"`),
	// Bearer tokens in Authorization headers
	regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9._\-]+`),
	// JWT tokens (eyJ...)
	regexp.MustCompile(`eyJ[a-zA-Z0-9_-]{20,}\.[a-zA-Z0-9_-]{20,}\.[a-zA-Z0-9_-]{20,}`),
}

// RedactString scrubs sensitive values from a string.
func RedactString(s string) string {
	for _, pat := range redactPatterns {
		s = pat.ReplaceAllStringFunc(s, func(match string) string {
			// Preserve the key name but replace the value
			return strings.ReplaceAll(match, match, "[REDACTED]")
		})
	}
	return s
}

// FilterEnv returns only whitelisted environment variables.
func FilterEnv() map[string]string {
	out := make(map[string]string)
	for _, kv := range os.Environ() {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		if IsEnvAllowed(parts[0]) {
			out[parts[0]] = parts[1]
		}
	}
	return out
}
