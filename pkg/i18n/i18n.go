// Package i18n provides the translate-on-write machinery: a pluggable
// Translator (stubbed by default, real MT provider slotted in via env — the
// same pattern as the payments gateway), plus Accept-Language parsing.
//
// Content is authored in a base language (English). On create/update of a
// translatable field, callers ask the Translator for each target language and
// persist the result with source='auto'. An admin/vendor edit stores
// source='manual', which auto-translation must never overwrite — so proper
// nouns (brand/dish names) can be corrected and stay corrected.
package i18n

import (
	"context"
	"log"
	"os"
	"strings"
)

// BaseLang is the language content is authored in.
const BaseLang = "en"

// Targets are the non-base languages auto-translation fills on write.
var Targets = []string{"ar"}

// Supported is the full set the API will localize responses into.
var Supported = map[string]bool{"en": true, "ar": true}

// Translator turns text from one language into another. Implementations must be
// safe for concurrent use.
type Translator interface {
	Translate(ctx context.Context, text, from, to string) (string, error)
}

// NoopTranslator returns the source text unchanged. It is the default when no
// machine-translation provider is configured: the translation infrastructure
// (columns, endpoints, Accept-Language serving, admin override) is fully live,
// and auto-filled values simply equal the base text until either a real
// provider is wired or an admin supplies a manual translation.
type NoopTranslator struct{}

func (NoopTranslator) Translate(_ context.Context, text, _, _ string) (string, error) {
	return text, nil
}

// NewTranslator selects a provider from the environment: Azure AI Translator
// when AZURE_TRANSLATOR_KEY is set, otherwise the NoopTranslator (which keeps
// the whole i18n pipeline live with auto-values equal to the base text). Env is
// already loaded by config.LoadConfig() before this is called.
func NewTranslator() Translator {
	if key := strings.TrimSpace(os.Getenv("AZURE_TRANSLATOR_KEY")); key != "" {
		log.Println("i18n: using Azure Translator")
		return newAzureTranslator(key)
	}
	log.Println("i18n: no MT provider configured — using Noop (auto == base text)")
	return NoopTranslator{}
}

// ParseLang resolves an Accept-Language header to a supported base tag,
// defaulting to BaseLang. It reads only the highest-priority tag's primary
// subtag (e.g. "ar-SA,en;q=0.8" -> "ar"); q-value ordering beyond the first
// tag is intentionally ignored — clients send their chosen UI language first.
func ParseLang(header string) string {
	header = strings.TrimSpace(header)
	if header == "" {
		return BaseLang
	}
	first := strings.SplitN(header, ",", 2)[0]
	first = strings.SplitN(first, ";", 2)[0]
	primary := strings.ToLower(strings.SplitN(strings.TrimSpace(first), "-", 2)[0])
	if Supported[primary] {
		return primary
	}
	return BaseLang
}
