package onpage

import "github.com/dazamate/go-json-schema/internal/vocabtool/audit"

func init() { audit.Register(UnmarkedBreadcrumb{}) }

// UnmarkedBreadcrumb warns about a breadcrumb nav that no BreadcrumbList
// marks up. It needs the page's HTML, on a page search engines index.
type UnmarkedBreadcrumb struct{}

// ID is the check's ID.
func (UnmarkedBreadcrumb) ID() string { return "onpage/unmarked-breadcrumb" }

// Run reports each breadcrumb nav without a BreadcrumbList.
func (UnmarkedBreadcrumb) Run(c *audit.Context) []audit.Issue { return analyse(c).breadcrumb }
