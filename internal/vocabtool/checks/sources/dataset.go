package sources

import (
	"github.com/dazamate/go-json-schema/internal/vocabtool/audit"
	"github.com/dazamate/go-json-schema/internal/vocabtool/txt"
	"github.com/dazamate/go-json-schema/internal/vocabtool/webpage"
)

func init() { audit.Register(Dataset{}) }

// Dataset warns on a block of data from a statistical source whose caption
// no value in the draft holds: the block is a Dataset.
type Dataset struct{}

// ID is the check's ID.
func (Dataset) ID() string { return "sources/dataset" }

// Run reports each block of statistics no Dataset names.
func (Dataset) Run(c *audit.Context) []audit.Issue {
	var out []audit.Issue
	for _, b := range dataBlocks(c) {
		if b.statistical() && !c.Index().HoldsText(b.caption) {
			out = append(out, audit.Issuef("html:"+webpage.SelectorFor(c.Page.Doc, b.n), audit.Warning, "the page shows data from %s (%q) but no Dataset describes it: add one (name and description from the caption and its note, creator or publisher the source, isBasedOn its URL, temporalCoverage from the years, spatialCoverage from the places, variableMeasured from the column headers)", txt.Clip(b.source, 100), txt.Clip(b.caption, 80)))
		}
	}
	return out
}
