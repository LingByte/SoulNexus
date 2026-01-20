package knowledge

// Config key constants
const (
	// Aliyun config keys
	ConfigKeyAliyunAccessKeyID     = "access_key_id"
	ConfigKeyAliyunAccessKeySecret = "access_key_secret"
	ConfigKeyAliyunEndpoint        = "endpoint"
	ConfigKeyAliyunWorkspaceID     = "workspace_id"
	ConfigKeyAliyunCategoryID      = "category_id"
	ConfigKeyAliyunSourceType      = "source_type"
	ConfigKeyAliyunParser          = "parser"
	ConfigKeyAliyunStructType      = "struct_type"
	ConfigKeyAliyunSinkType        = "sink_type"

	// Milvus config keys
	ConfigKeyMilvusAddress        = "address"
	ConfigKeyMilvusUsername       = "username"
	ConfigKeyMilvusPassword       = "password"
	ConfigKeyMilvusCollectionName = "collection_name"
	ConfigKeyMilvusDimension      = "dimension"

	// Qdrant config keys
	ConfigKeyQdrantBaseURL        = "base_url"
	ConfigKeyQdrantApiKey         = "api_key"
	ConfigKeyQdrantCollectionName = "collection_name"
	ConfigKeyQdrantDimension      = "dimension"

	// Elasticsearch config keys
	ConfigKeyElasticsearchBaseURL   = "base_url"
	ConfigKeyElasticsearchUsername  = "username"
	ConfigKeyElasticsearchPassword  = "password"
	ConfigKeyElasticsearchIndexName = "index_name"

	// Pinecone config keys
	ConfigKeyPineconeApiKey    = "api_key"
	ConfigKeyPineconeBaseURL   = "base_url"
	ConfigKeyPineconeIndexName = "index_name"
	ConfigKeyPineconeDimension = "dimension"
)

// Default value constants
const (
	// Aliyun default values
	DefaultAliyunEndpoint   = "bailian.cn-beijing.aliyuncs.com"
	DefaultAliyunSourceType = "DATA_CENTER_FILE"
	DefaultAliyunParser     = "DASHSCOPE_DOCMIND"
	DefaultAliyunStructType = "unstructured"
	DefaultAliyunSinkType   = "BUILT_IN"

	// Milvus default values
	DefaultMilvusAddress   = "localhost:19530"
	DefaultMilvusDimension = 768

	// Pinecone default values
	DefaultPineconeBaseURL   = "https://api.pinecone.io"
	DefaultPineconeDimension = 1536
)

// Metadata key constants
const (
	MetadataKeyUserID = "user_id"
	MetadataKeyName   = "name"
	MetadataKeySource = "source"
)

// Metadata value constants
const (
	MetadataSourceAPICreate = "api_create"
	MetadataSourceAPIUpload = "api_upload"
)

// Knowledge base name separator
const KnowledgeNameSeparator = "-"

// Error message constants
const (
	ErrKnowledgeBaseDisabled    = "knowledge base feature is disabled, please set KNOWLEDGE_BASE_ENABLED=true in config"
	ErrUnsupportedProvider      = "unsupported knowledge base provider: %s"
	ErrAccessKeyRequired        = "access_key_id and access_key_secret are required"
	ErrApiKeyRequired           = "api_key is required"
	ErrCollectionNameRequired   = "collection_name is required"
	ErrIndexNameRequired        = "index_name is required"
	ErrKnowledgeKeyRequired     = "knowledge key cannot be empty"
	ErrKnowledgeNotFound        = "knowledge base not found"
	ErrConfigParseFailed        = "failed to parse config"
	ErrKnowledgeBaseInitFailed  = "failed to initialize knowledge base instance"
	ErrFileReceiveFailed        = "failed to receive file"
	ErrFileEmpty                = "file cannot be empty"
	ErrFileUploadFailed         = "failed to upload file"
	ErrIndexCreateFailed        = "failed to create index"
	ErrIndexDeleteFailed        = "failed to delete knowledge base"
	ErrDatabaseDeleteFailed     = "failed to delete database record"
	ErrQueryKnowledgeListFailed = "failed to query knowledge base list"
)
