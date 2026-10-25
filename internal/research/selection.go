package research

import "encoding/json"

// Selected is one entity chosen for about or mentions: its identity, the
// scores that qualified it and its ready schema.org node.
type Selected struct {
	Name      string `json:"name"`
	QID       string `json:"qid"`
	Source    string `json:"source"`
	Wikipedia string `json:"wikipedia"`

	KGMID        string `json:"kgmid,omitempty"`
	GoogleKGID   string `json:"google_kg_id,omitempty"`
	KGMIDChecked bool   `json:"kgmid_checked"`
	BingID       string `json:"bing_id,omitempty"`
	BingChecked  bool   `json:"bing_checked"`

	Relevance         *float64 `json:"relevance,omitempty"`
	Topic             *float64 `json:"topic,omitempty"`
	Confidence        *float64 `json:"confidence,omitempty"`
	KeywordSimilarity *float64 `json:"keyword_similarity,omitempty"`
	PageSimilarity    *float64 `json:"page_similarity,omitempty"`
	Mentions          int      `json:"mentions,omitempty"`

	Matched     []string `json:"matched"`
	LinkedTo    []string `json:"linked_to"`
	QualifiedBy []string `json:"qualified_by"`

	Thing json.RawMessage `json:"thing"`
}
