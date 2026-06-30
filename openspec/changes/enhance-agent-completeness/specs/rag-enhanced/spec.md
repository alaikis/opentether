## ADDED Requirements

### Requirement: System includes built-in embedding model
The system SHALL embed a lightweight ONNX embedding model (bge-micro-v2, ~50MB) that runs locally without external API calls.

#### Scenario: Embedding model loads
- **WHEN** the service starts
- **THEN** the system SHALL load the ONNX model into memory within 5 seconds

#### Scenario: Generate embeddings locally
- **WHEN** user requests text embedding
- **THEN** the system SHALL use the local model to generate embeddings without calling external APIs

### Requirement: System supports incremental index updates
The system SHALL allow adding new documents to the existing index without full rebuild.

#### Scenario: Add document to index
- **WHEN** user calls POST /api/v1/rag/documents with a new document
- **THEN** the system SHALL update the vector index with the new document within 1 second

#### Scenario: Batch document addition
- **WHEN** user calls POST /api/v1/rag/documents/batch with multiple documents
- **THEN** the system SHALL bulk-insert all documents and confirm completion

### Requirement: System supports hybrid search
The system SHALL combine keyword and semantic search for improved relevance.

#### Scenario: Hybrid search
- **WHEN** user submits a query
- **THEN** the system SHALL compute both BM25 keyword score and semantic similarity, then combine using weighted ranking

#### Scenario: Configurable hybrid weights
- **WHEN** user sets rag.hybrid_weight in config
- **THEN** the system SHALL use the specified ratio (e.g., 0.7 semantic + 0.3 keyword)

### Requirement: Document metadata filtering
The system SHALL support filtering documents by metadata during retrieval.

#### Scenario: Filter by metadata
- **WHEN** query includes filter conditions (e.g., category="sales")
- **THEN** the system SHALL exclude documents not matching the filter from results