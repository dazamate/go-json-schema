//go:build cgo

package embed

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/sugarme/tokenizer"
	"github.com/sugarme/tokenizer/normalizer"
	"github.com/sugarme/tokenizer/pretrained"
	ort "github.com/yalue/onnxruntime_go"
)

// The pinned downloads behind Model, saved under the dir given to NewONNX.
const modelBase = "https://huggingface.co/Qdrant/bge-base-en-v1.5-onnx-Q/resolve/199291fdd1aa89faf9c20b722dc72ad5e17aa0d0/"

var (
	modelAsset = asset{
		name:   "bge-base-en-v1.5-onnx-Q@199291f/model_optimized.onnx",
		url:    modelBase + "model_optimized.onnx",
		sha256: "4e556722bc4f65716c544c8a931f1e90fb3f866e5741fd93a96f051d673339c7",
	}
	tokenizerAsset = asset{
		name:   "bge-base-en-v1.5-onnx-Q@199291f/tokenizer.json",
		url:    modelBase + "tokenizer.json",
		sha256: "d241a60d5e8f04cc1b2b3e9ef7a4921b27bf526d9f6050ab90f9267a1f9e5c66",
	}
	// runtimeAsset is ONNX Runtime 1.31.0, the C API onnxruntime_go v1.38.0
	// is built against. Only linux/amd64 has a pinned download.
	runtimeAsset = asset{
		name:    "onnxruntime-1.31.0/libonnxruntime.so.1.31.0",
		url:     "https://github.com/microsoft/onnxruntime/releases/download/v1.31.0/onnxruntime-linux-x64-1.31.0.tgz",
		archive: "cc5c72baf5ae5c8238a6841f0897227be2d02826b9cf98eaf02fdefaeeb03a57",
		member:  "onnxruntime-linux-x64-1.31.0/lib/libonnxruntime.so.1.31.0",
		sha256:  "469849977cb60744ea160927e1a3a0643fd4c55c0aff743251c6d293402e334e",
	}
)

const hiddenSize = 768 // last_hidden_state's width

// ONNX runs Model locally through ONNX Runtime. It downloads its files on the
// first Encode.
type ONNX struct {
	dir      string
	progress func(ctx context.Context, msg string)

	mu      sync.Mutex
	tok     *tokenizer.Tokenizer
	session *ort.DynamicAdvancedSession
}

// NewONNX returns an encoder that keeps its files in dir. Nothing loads until
// the first Encode. progress, which may be nil, hears about downloads.
func NewONNX(dir string, progress func(ctx context.Context, msg string)) *ONNX {
	return &ONNX{dir: dir, progress: progress}
}

// The ONNX Runtime environment is process-wide, so it is set up once.
var (
	ortOnce sync.Once
	ortErr  error
)

func initRuntime(lib string) error {
	ortOnce.Do(func() {
		ort.SetSharedLibraryPath(lib)
		ortErr = ort.InitializeEnvironment()
	})
	return ortErr
}

// load readies the tokenizer and the session. The caller holds o.mu. A failed
// load is tried again on the next call.
func (o *ONNX) load(ctx context.Context) error {
	if o.session != nil {
		return nil
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("no embedding runtime download for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	report := func(msg string) {
		if o.progress != nil {
			o.progress(ctx, msg)
		}
	}
	if err := ensure(ctx, o.dir, []asset{modelAsset, tokenizerAsset, runtimeAsset}, report); err != nil {
		return err
	}
	if err := initRuntime(filepath.Join(o.dir, filepath.FromSlash(runtimeAsset.name))); err != nil {
		return fmt.Errorf("embed: starting ONNX Runtime: %w", err)
	}
	tok, err := pretrained.FromFile(filepath.Join(o.dir, filepath.FromSlash(tokenizerAsset.name)))
	if err != nil {
		return fmt.Errorf("embed: loading the tokenizer: %w", err)
	}
	tok.WithTruncation(&tokenizer.TruncationParams{MaxLength: MaxTokens, Strategy: tokenizer.LongestFirst})
	tok.WithPadding(nil) // one text per run
	// sugarme reads the file's "strip_accents": null as false; Hugging Face
	// reads it as "follow lowercase", which is true. sugarme's strip also only
	// drops combining marks without decomposing first, so a precomposed "é"
	// would survive it: NFD runs first, as in Hugging Face's strip. The accent
	// probe in TestONNXMatchesPythonReference guards both.
	tok.WithNormalizer(normalizer.NewSequence([]normalizer.Normalizer{
		normalizer.NewNFD(),
		normalizer.NewBertNormalizer(true, true, true, true), // clean text, lowercase, Chinese chars, strip accents
	}))
	session, err := ort.NewDynamicAdvancedSession(filepath.Join(o.dir, filepath.FromSlash(modelAsset.name)),
		[]string{"input_ids", "attention_mask", "token_type_ids"}, []string{"last_hidden_state"}, nil)
	if err != nil {
		return fmt.Errorf("embed: loading the model: %w", err)
	}
	o.tok, o.session = tok, session
	return nil
}

func (o *ONNX) encoding(ctx context.Context, text string) (*tokenizer.Encoding, error) {
	if err := o.load(ctx); err != nil {
		return nil, err
	}
	enc, err := o.tok.EncodeSingle(text, true)
	if err != nil {
		return nil, fmt.Errorf("embed: tokenizing: %w", err)
	}
	return enc, nil
}

// tokens is the model's token ids for text.
func (o *ONNX) tokens(ctx context.Context, text string) ([]int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	enc, err := o.encoding(ctx, text)
	if err != nil {
		return nil, err
	}
	return enc.Ids, nil
}

// Encode embeds text alone, with no padding, truncated to MaxTokens: the
// CLS token's hidden state, normalised.
func (o *ONNX) Encode(ctx context.Context, text string) ([]float64, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	enc, err := o.encoding(ctx, text)
	if err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	n := int64(len(enc.Ids))
	var inputs []ort.Value
	defer func() {
		for _, v := range inputs {
			v.Destroy()
		}
	}()
	for _, col := range [][]int{enc.Ids, enc.AttentionMask, enc.TypeIds} {
		data := make([]int64, n)
		for i, x := range col {
			data[i] = int64(x)
		}
		t, err := ort.NewTensor(ort.NewShape(1, n), data)
		if err != nil {
			return nil, fmt.Errorf("embed: %w", err)
		}
		inputs = append(inputs, t)
	}
	out, err := ort.NewEmptyTensor[float32](ort.NewShape(1, n, hiddenSize))
	if err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}
	defer out.Destroy()
	if err := o.session.Run(inputs, []ort.Value{out}); err != nil {
		return nil, fmt.Errorf("embed: %w", err)
	}
	cls := make([]float64, hiddenSize)
	for i, x := range out.GetData()[:hiddenSize] {
		cls[i] = float64(x)
	}
	return unit(cls), nil
}

// Close frees the session. The ONNX Runtime environment stays up for the
// process.
func (o *ONNX) Close() error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.session == nil {
		return nil
	}
	err := o.session.Destroy()
	o.tok, o.session = nil, nil
	return err
}
