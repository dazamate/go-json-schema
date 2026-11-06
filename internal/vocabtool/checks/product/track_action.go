package product

import (
	"regexp"

	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
)

func init() { audit.Register(TrackAction{}) }

var trackOrder = regexp.MustCompile(`(?i)^track (?:your |my |an )?order`)

// TrackAction warns when a page links order tracking and the draft has no
// TrackAction: the link is an action of the business.
type TrackAction struct{}

// ID is the check's ID.
func (TrackAction) ID() string { return "product/track-action" }

// Run reports the first track-your-order link or button, at its element.
func (TrackAction) Run(c *audit.Context) []audit.Issue {
	l := engagementLink(c, kindTrack)
	if l == nil {
		return nil
	}
	return []audit.Issue{audit.Issuef(l.path, audit.Warning, "the page links order tracking (%q): add a TrackAction (target that page) to the business's potentialAction", l.text)}
}
