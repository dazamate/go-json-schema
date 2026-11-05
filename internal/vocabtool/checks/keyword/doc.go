// Package keyword holds the checks of the draft against the page's target
// keyword, and the checks of who the page is about. The page's own name,
// headline, description and keywords properties must carry the keyword, every
// description on its own content nodes must reach the target density, and so
// must the markup's text as a whole. The keyword checks run only when the
// caller gives a keyword. Beside them, an organization in about and a
// business in mentions read as if the page were about that business, and a
// flat graph lists every node it points at as an item of its own. These
// checks read the page's facts, or only its URL when there is no page, and
// hold a page search engines skip to them too.
package keyword
