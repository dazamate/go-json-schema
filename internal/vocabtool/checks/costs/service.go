package costs

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Service{}) }

// serviceCost reads what an h1 says a service costs, capturing the service
// and its head word: the cost to do a job ("Cost to tile a bathroom"), of
// doing it ("The cost of plastering a room"), or what its provider or the job
// charges ("How much do plumbers charge?", "Landscaping costs", "Kitchen
// renovation costs"). notService drops the heads that name no service.
var serviceCost = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bcosts?\s+to\s+((\w+)\b[^,:?(]*)`),
	regexp.MustCompile(`(?i)\bcosts?\s+of\s+((\w+ing)\b[^,:?(]*)`),
	regexp.MustCompile(`(?i)^(?:how much (?:do|does|is|are) |what (?:do|does|is|are) )?(?:the |an? |\d+ )?((?:\w+ )?(\w+ers|\w+ing|\w+ations?|company|companies|crew|services?|labou?r))\s+(?:cost|costs|charge)\b`),
}

// notService are the head words that follow "cost to" or come before "costs"
// without naming a service: whom it costs ("The cost to your business"), or
// an expense ("Heating costs", "The cost of living", "Car running costs").
var notService = regexp.MustCompile(`(?i)^(?:the|a|an|you|your|us|our|them|their|me|my|him|her|it|its|consumers|customers|users|others|owners|taxpayers|business|businesses|own|live|run|operate|buy|park|living|heating|cooling|running|housing|operating|closing|parking|lighting|funding|financing|borrowing|wedding|clothing|schooling|dining|eating|commuting|spending|owning|buying|selling|nothing|something|anything|everything|thing|ceiling|morning|evening|inflation|taxation)$`)

// Service warns on a content page of a site that sells nothing whose h1 is
// about what a service costs ("Cost to move a piano", "How much do movers
// cost?") when no Service describes it.
type Service struct{}

// ID is the check's ID.
func (Service) ID() string { return "costs/service" }

// Run reports the service the h1 names.
func (Service) Run(c *audit.Context) []audit.Issue {
	if !costPage(c) || c.Page.Kind != webpage.KindOther || c.Has("Service") {
		return nil
	}
	h := c.Page.H1()
	for _, re := range serviceCost {
		if m := re.FindStringSubmatch(h); m != nil && !notService.MatchString(m[2]) {
			return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page is about the cost of the service %q but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity", strings.TrimSpace(m[1]))}
		}
	}
	return nil
}
