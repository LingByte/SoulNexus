package models

// Ollama vision / multimodal models.
const (
	OllamaLLaVA       = "llava"
	OllamaLLaVALatest = "llava:latest"
	OllamaLLaVA_7B    = "llava:7b"
	OllamaLLaVA_13B   = "llava:13b"
	OllamaLLaVA_34B   = "llava:34b"

	OllamaLLaVAPhi3   = "llava-phi3"
	OllamaLLaVALlama3 = "llava-llama3"
	OllamaBakLLaVA    = "bakllava"
	OllamaMoondream   = "moondream"
	OllamaMiniCPMV    = "minicpm-v"
)

// Ollama embedding models.
const (
	OllamaNomicEmbedText       = "nomic-embed-text"
	OllamaMXBAIEmbedLarge      = "mxbai-embed-large"
	OllamaSnowflakeArcticEmbed = "snowflake-arctic-embed"
)

// Ollama Cohere, IBM, and other vendor models.
const (
	OllamaCommandR         = "command-r"
	OllamaCommandRLatest   = "command-r:latest"
	OllamaCommandR7B       = "command-r7b"
	OllamaCommandRPlus     = "command-r-plus"
	OllamaCommandRPlus104B = "command-r-plus:104b"

	OllamaGranite3       = "granite3"
	OllamaGranite3Latest = "granite3:latest"
	OllamaGranite3_1B    = "granite3-dense:1b"
	OllamaGranite3_2B    = "granite3-dense:2b"
	OllamaGranite3_8B    = "granite3-dense:8b"

	OllamaSolar       = "solar"
	OllamaSolarLatest = "solar:latest"
	OllamaSolar10_7B  = "solar:10.7b"

	OllamaYi       = "yi"
	OllamaYiLatest = "yi:latest"
	OllamaYi_6B    = "yi:6b"
	OllamaYi_9B    = "yi:9b"
	OllamaYi_34B   = "yi:34b"

	OllamaNousHermes2            = "nous-hermes2"
	OllamaNousHermes2Latest      = "nous-hermes2:latest"
	OllamaNousHermes2Mixtral     = "nous-hermes2-mixtral"
	OllamaNousHermes2Mixtral8x7B = "nous-hermes2-mixtral:8x7b"

	OllamaWizardLM2       = "wizardlm2"
	OllamaWizardLM2_7B    = "wizardlm2:7b"
	OllamaWizardLM2_8x22B = "wizardlm2:8x22b"

	OllamaDolphinMixtral      = "dolphin-mixtral"
	OllamaDolphinMixtral8x7B  = "dolphin-mixtral:8x7b"
	OllamaDolphinMixtral8x22B = "dolphin-mixtral:8x22b"

	OllamaOpenChat       = "openchat"
	OllamaOpenChatLatest = "openchat:latest"
	OllamaOpenChat7B     = "openchat:7b"

	OllamaVicuna       = "vicuna"
	OllamaVicunaLatest = "vicuna:latest"
	OllamaVicuna_7B    = "vicuna:7b"
	OllamaVicuna_13B   = "vicuna:13b"
	OllamaVicuna_33B   = "vicuna:33b"

	OllamaOrcaMini   = "orca-mini"
	OllamaOrcaMini3B = "orca-mini:3b"
	OllamaOrcaMini7B = "orca-mini:7b"

	OllamaTinyLlama    = "tinyllama"
	OllamaSmolLM2      = "smollm2"
	OllamaSmolLM2_135M = "smollm2:135m"
	OllamaSmolLM2_360M = "smollm2:360m"
	OllamaSmolLM2_1_7B = "smollm2:1.7b"
)
