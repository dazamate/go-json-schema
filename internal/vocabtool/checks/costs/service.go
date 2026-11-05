package costs

import (
	"regexp"
	"strings"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Service{}) }

// serviceCost reads what an h1 says a service costs. The second pattern names
// the service by a provider ("movers", "cleaning company") or by the gerund of
// a service verb ("Moving costs"); any other gerund ("Heating costs", "Car
// running costs") is an expense, not a service.
var serviceCost = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bcosts?\s+(?:of|to)\s+((?:move|moving|hire|hiring|install|installing|repair|repairing|clean|cleaning|replace|replacing|remove|removing|fix|fixing|build|building|paint|painting|ship|shipping|store|storing|haul|hauling|pack|packing|deliver|delivering)\b[^,:?(]*)`),
	regexp.MustCompile(`(?i)^(?:how much (?:do|does|is|are) |what (?:do|does|is|are) )?(?:the |an? |\d+ )?((?:\w+ )?(?:\w+ers|moving|hiring|installing|installation|repairing|cleaning|replacing|removing|fixing|building|painting|shipping|storing|hauling|packing|delivering|company|companies|crew|services?|labou?r))\s+(?:cost|costs|charge)\b`),
}

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
		if m := re.FindStringSubmatch(h); m != nil {
			return []audit.Issue{audit.Issuef(audit.PagePath, audit.Warning, "the page is about the cost of the service %q but no Service describes it: add one (name, serviceType, areaServed, offers or priceSpecification for the prices shown, provider when it is the business's own) and make it the WebPage's about or mainEntity", strings.TrimSpace(m[1]))}
		}
	}
	return nil
}
