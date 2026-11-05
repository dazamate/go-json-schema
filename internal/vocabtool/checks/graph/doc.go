// Package graph holds the checks of the draft's nodes against one another
// and against what they say of themselves. A node can be typed more
// specifically than it is: its first Wikidata class can map to a subtype of
// its type (a Place whose class is city: City), or its name, headline or URL
// path can name one (a WebPage at /about/: AboutPage). Only stable types are
// suggested: not pending, retired, meta, superseded, a data type or an
// enumeration. The package also holds the checks of a node's own shape: a
// node that repeats another's Wikidata item, a bare category type, a generic
// WebPageElement, a Review, an Offer or a Person that leaves out what it
// names, a WebPage with no url or name, and the reviews of the page's own
// publisher. The last three read the page's facts, and run on a page search
// engines skip too.
package graph
