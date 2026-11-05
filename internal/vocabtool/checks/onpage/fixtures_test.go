package onpage_test

// elementsPage has one of each section a specific type describes, plus the
// ones that must be skipped: an article's own header, a presentation table,
// a template, a hidden nav and a script.
const elementsPage = `<!doctype html><html><head><title>Run sheet</title><style>nav{}</style></head><body>
<header id="top"><nav id="site-nav" aria-label="Main"><a href="/">Home</a></nav></header>
<nav aria-label="Breadcrumb"><ol><li>Home</li></ol></nav>
<main><article><header><h1>Banquet run sheet</h1></header>
<table><tr><td>6pm</td><td>Doors open</td></tr></table>
<table role="presentation"><tr><td>layout</td></tr></table>
<aside><h2>Related guides</h2><p>Tea ceremony</p></aside>
<ins class="adsbygoogle" data-ad-slot="1"></ins>
</article></main>
<footer><nav><a href="/contact/">Contact</a></nav><nav><a href="/privacy/">Privacy</a></nav></footer>
<template><nav id="ghost"></nav></template>
<nav hidden id="hidden-nav"></nav>
<script>var x = "<nav>Kitchen</nav>";</script>
</body></html>`

// footerPage has one section a specific type describes: its footer.
const footerPage = `<!doctype html><html><head><title>Tea</title></head><body><main><h1>Tea</h1><p>Steps and order.</p></main><footer><p>Contact us</p></footer></body></html>`

// crumbPage has a breadcrumb nav and nothing else a specific type describes.
const crumbPage = `<!doctype html><html><head><title>Tea</title></head><body><nav aria-label="Breadcrumb"><ol><li>Home</li></ol></nav><main><h1>Tea</h1><p>Steps and order.</p></main></body></html>`

// teaPage is a page of no particular kind, for the checks that need a page
// but read none of its sections.
const teaPage = `<!doctype html><html><head><title>Tea ceremony</title></head><body><main><h1>Tea ceremony</h1><h2 id="step-1">Boil</h2><p>Steps and order.</p></main></body></html>`
