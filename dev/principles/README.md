# Development Principles

## 1. Signal over completeness
Optimize for precision and low noise, not exhaustive detection. A feature that finds more events but materially increases false positives is a regression.

## 2. Evidence over inference
Every result must originate from observable repository or work-item facts and retain provenance back to them. Detect architectural signals, not architectural correctness.

## 3. Explicit architecture over discovered architecture
Treat intended systems, bounded contexts, components, and significant artifacts as declared configuration. Do not reconstruct architecture through expensive dependency or semantic analysis unless proven necessary.

## 4. Minimum sufficient analysis
Use the cheapest reliable mechanism capable of answering the question. Prefer metadata → paths → change sets → rules. Add dependency analysis, AST analysis, embeddings, or LLMs only when a demonstrated detection gap justifies the additional complexity.
