"""Records what the Go embedder must reproduce: google-seo-docs' recipe
(seoaudit/embed.py, fastembed with Qdrant/bge-base-en-v1.5-onnx-Q) on the
calibration page, at full precision. Re-run only when the calibration data
or the model changes:

  /home/dazamate/projects/google-seo-docs/.venv/bin/python -I internal/embed/testdata/record_reference.py
"""
import json
import sys
from importlib.metadata import version
from pathlib import Path

sys.path.insert(0, "/home/dazamate/projects/google-seo-docs")
from seoaudit.embed import QUERY_INSTRUCTION, Embedder, chunks, mean_vector  # noqa: E402

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[2]
KEYWORD = "moving cost calculator"
PROBES = ["Café crème brûlée, naïve façade", " ".join(["moving"] * 600)]

page = (HERE / "page.txt").read_text()
tr = json.loads((ROOT / "internal/research/testdata/select/textrazor.json").read_text())["response"]
labels = sorted({t["label"] for t in tr["topics"] if t.get("wikidataId")})
emb = Embedder()
tok = emb._load().model.tokenizer
ids = lambda text: tok.encode(text).ids
dot = lambda a, b: sum(x * y for x, y in zip(a, b))

query = QUERY_INSTRUCTION + KEYWORD
kv = emb.embed([query])[0]
windows = chunks(page)
pv = mean_vector(emb.embed(windows))
themes = []
for label in labels:
    v = emb.embed([label])[0]
    themes.append({"label": label, "ids": ids(label), "keyword": dot(kv, v), "page": dot(pv, v)})
ref = {"model": "BAAI/bge-base-en-v1.5", "source": "Qdrant/bge-base-en-v1.5-onnx-Q",
       "revision": "199291fdd1aa89faf9c20b722dc72ad5e17aa0d0", "fastembed": version("fastembed"),
       "onnxruntime": version("onnxruntime"), "keyword": KEYWORD,
       "query": {"text": query, "ids": ids(query)},
       "windows": [{"text": w, "ids": ids(w)} for w in windows],
       "probes": [{"text": p, "ids": ids(p)} for p in PROBES], "themes": themes}
(HERE / "reference.json").write_text(json.dumps(ref, indent=1, ensure_ascii=False) + "\n")
cos = {"model": ref["model"], "keyword": KEYWORD,
       "themes": {t["label"]: {"keyword": t["keyword"], "page": t["page"]} for t in themes}}
(ROOT / "internal/research/testdata/select/cosines.json").write_text(json.dumps(cos, indent=1, sort_keys=True) + "\n")
print(f"{len(windows)} windows, {len(themes)} themes")
