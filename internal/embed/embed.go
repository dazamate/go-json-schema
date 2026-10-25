// Package embed is the text-similarity recipe behind select_entities. It
// follows seoaudit/embed.py in google-seo-docs, and the cutoffs in
// select_entities were set with exactly this recipe, so it must not change.
package embed

import (
	"context"
	"errors"
	"math"
	"strings"
)

// Model names the exact build the cutoffs were calibrated against.
const Model = "BAAI/bge-base-en-v1.5 (Qdrant/bge-base-en-v1.5-onnx-Q@199291f)"

// QueryInstruction is BGE's instruction for short search queries; passages take no prefix.
const QueryInstruction = "Represent this sentence for searching relevant passages: "

const (
	WindowWords  = 200 // about 270 tokens, well inside the model's 512
	WindowStride = 150 // 50 words of overlap, so a passage cut at a window edge is whole in the next
	MaxWindows   = 40  // about 6,000 words of main text per page
	MaxTokens    = 512 // the model's input limit
)

// Encoder turns text into a unit vector.
type Encoder interface {
	Encode(ctx context.Context, text string) ([]float64, error)
}

// Windows splits text into overlapping word windows, in order.
func Windows(text string) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}
	end := max(1, len(words)-WindowWords+WindowStride)
	var out []string
	for i := 0; i < end && len(out) < MaxWindows; i += WindowStride {
		out = append(out, strings.Join(words[i:min(i+WindowWords, len(words))], " "))
	}
	return out
}

func unit(v []float64) []float64 {
	var sum float64
	for _, x := range v {
		sum += x * x
	}
	norm := math.Sqrt(sum)
	if norm == 0 {
		norm = 1
	}
	out := make([]float64, len(v))
	for i, x := range v {
		out[i] = x / norm
	}
	return out
}

// Mean is the normalised mean of unit vectors: one vector for a page.
func Mean(vs [][]float64) []float64 {
	if len(vs) == 0 {
		return nil
	}
	mean := make([]float64, len(vs[0]))
	for _, v := range vs {
		for i, x := range v {
			mean[i] += x
		}
	}
	for i := range mean {
		mean[i] /= float64(len(vs))
	}
	return unit(mean)
}

// Cosine is the cosine similarity of two unit vectors, not rounded.
func Cosine(a, b []float64) float64 {
	var sum float64
	for i := 0; i < len(a) && i < len(b); i++ {
		sum += a[i] * b[i]
	}
	return sum
}

// Scorer scores labels against a keyword and a page.
type Scorer struct {
	Enc Encoder
}

// Model names the model behind the scores.
func (s *Scorer) Model() string { return Model }

// Score returns each label's cosine against the keyword (embedded with
// QueryInstruction) and against the page (the Mean of its Windows). Labels are
// embedded alone, with no instruction.
func (s *Scorer) Score(ctx context.Context, keyword, page string, labels []string) (kw, pg []float64, err error) {
	windows := Windows(page)
	if len(windows) == 0 {
		return nil, nil, errors.New("embed: the page has no words")
	}
	query, err := s.Enc.Encode(ctx, QueryInstruction+keyword)
	if err != nil {
		return nil, nil, err
	}
	vecs := make([][]float64, len(windows))
	for i, w := range windows {
		if vecs[i], err = s.Enc.Encode(ctx, w); err != nil {
			return nil, nil, err
		}
	}
	doc := Mean(vecs)
	for _, label := range labels {
		v, err := s.Enc.Encode(ctx, label)
		if err != nil {
			return nil, nil, err
		}
		kw = append(kw, Cosine(v, query))
		pg = append(pg, Cosine(v, doc))
	}
	return kw, pg, nil
}
