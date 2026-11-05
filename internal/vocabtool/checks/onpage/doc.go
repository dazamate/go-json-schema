// Package onpage holds the checks that hold the draft to the page it marks
// up. The sections of the page (its header, footer, navs, sidebars, tables,
// ad slots and breadcrumb) each need a node that marks them up, and a
// node's cssSelector must select a section of its type. The about and
// mentions entities, the question, answer, step and review text, and the
// links to this page's own anchors must be on the page. Page elements, the
// sections with the node that marks each up, are the fact the section checks
// share, and Check reports them too. The text and anchor checks and the
// entity check run on a page search engines skip; the section checks ask
// nothing of one, though it still lists its sections.
package onpage
