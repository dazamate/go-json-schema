// Package graph holds the checks of the draft's nodes against one another
// and against what they say of themselves. A node can be typed more
// specifically than it is: its first Wikidata class can map to a subtype of
// its type (a Place whose class is city: City), or its name, headline or URL
// path can name one (a WebPage at /about/: AboutPage). Only stable types are
// suggested: not pending, retired, meta, superseded, a data type or an
// enumeration.
package graph
