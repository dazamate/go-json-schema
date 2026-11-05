package vocabtool

import (
	"encoding/json"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	_ "github.com/dazamate/go-json-schema/internal/vocabtool/checks/all" // register every check
	"github.com/dazamate/go-json-schema/internal/vocabtool/checks/onpage"
	"github.com/dazamate/go-json-schema/internal/vocabtool/jsonld"
	"github.com/dazamate/go-json-schema/internal/vocabtool/vocab"
)

// Issue severities. Only errors stop a draft being valid schema.org.
const (
	Error   = audit.Error
	Warning = audit.Warning
	Info    = audit.Info
)

// Issue is one finding about a draft, located by a JSONPath-like path, or for
// a finding on the page itself, by "html:" and the element's CSS selector.
type Issue = audit.Issue

// Report is the checked draft, ready to publish, plus what was found.
type Report struct {
	Errors       int           `json:"errors"`
	Warnings     int           `json:"warnings"`
	Issues       []Issue       `json:"issues"`
	PageElements []PageElement `json:"page_elements,omitempty"`
	ScriptTag    string        `json:"script_tag"`
}

// PageElement is a section of the page that a specific schema.org type
// describes: a WebPageElement subtype, or a BreadcrumbList for a breadcrumb
// nav. MarkedBy is the path of the draft node that marks it up, "" for none.
type PageElement = onpage.PageElement

// CheckOptions controls the published form and the page checks.
type CheckOptions struct {
	Pretty   bool   // indent the JSON inside the script tag
	PageText string // the page's visible text; when set, about and mentions entities must be named in it
	PageHTML string // the rendered page; its sections must be marked up, and it gives PageText when that is empty
	PageURL  string // the page's address; links resolve against it when the page has no canonical link
	// Keyword is the page's target keyword; when set, the page's names,
	// headlines and descriptions must carry it at KeywordDensity.
	Keyword string
	// KeywordDensity is the keyword's target share of the words, in percent;
	// 0 is DefaultKeywordDensity.
	KeywordDensity float64
}

// Check validates a JSON-LD draft against the vocabulary and returns it as
// a script tag. Unknown types, properties and enumeration members are
// errors. A property used on a type outside its listed domains is only
// info: schema.org calls domains "closer to guidelines than formal rules"
// (https://schema.org/docs/datamodel.html), so it is kept. Bare enum member
// names such as InStock are expanded to their schema.org URL.
func (t *Tool) Check(draft json.RawMessage, opts CheckOptions) (Report, error) {
	ctx, err := audit.NewContext(t.Vocab, draft, input(opts))
	if err != nil {
		return Report{}, err
	}
	rep := Report{Issues: audit.Run(ctx, audit.Checks())}
	if rep.Issues == nil {
		rep.Issues = []Issue{}
	}
	rep.PageElements = onpage.Elements(ctx)
	for _, i := range rep.Issues {
		switch i.Severity {
		case Error:
			rep.Errors++
		case Warning:
			rep.Warnings++
		}
	}
	if rep.ScriptTag, err = scriptTag(ctx.Root, opts.Pretty); err != nil {
		return Report{}, err
	}
	return rep, nil
}

// input is what the audit is given beside the draft.
func input(opts CheckOptions) audit.Input {
	return audit.Input{
		PageText:       opts.PageText,
		PageHTML:       opts.PageHTML,
		PageURL:        opts.PageURL,
		Keyword:        opts.Keyword,
		KeywordDensity: opts.KeywordDensity,
	}
}

// scriptTag is the draft as Check publishes it: a JSON-LD script tag.
func scriptTag(root any, pretty bool) (string, error) {
	body, err := jsonld.Encode(root, pretty)
	if err != nil {
		return "", err
	}
	safe := strings.NewReplacer("<", `<`, ">", `>`).Replace(string(body))
	return `<script type="application/ld+json">` + safe + `</script>`, nil
}

// Vocabulary checks a draft against the vocabulary alone: the terms checks
// Check runs too, without its page, site and naming checks. Unknown types,
// properties and enumeration members are errors; superseded terms, an
// untyped node with properties and a property only a subtype takes are
// warnings; a property outside its listed domains is info.
func (t *Tool) Vocabulary(draft json.RawMessage) ([]Issue, error) {
	ctx, err := audit.NewVocabularyContext(t.Vocab, draft)
	if err != nil {
		return nil, err
	}
	return audit.Run(ctx, audit.Group("terms/")), nil
}

// checker makes the audit's checks that have not moved to their own files
// yet. The legacy adapter check runs it.
type checker struct {
	v      *vocab.Vocab
	issues []Issue
}

// newChecker is a checker with nothing found yet.
func newChecker(v *vocab.Vocab) *checker {
	return &checker{v: v}
}

// add notes an issue.
func (c *checker) add(path, severity, format string, args ...any) {
	c.issues = append(c.issues, audit.Issuef(path, severity, format, args...))
}
