package models

// Mistral AI flagship models.
const (
	MistralLargeLatest  = "mistral-large-latest"
	MistralLarge2411    = "mistral-large-2411"
	MistralLarge2407    = "mistral-large-2407"
	MistralSmallLatest  = "mistral-small-latest"
	MistralSmall2501    = "mistral-small-2501"
	MistralSmall2409    = "mistral-small-2409"
	MistralMediumLatest = "mistral-medium-latest"
	MistralMedium2312   = "mistral-medium-2312"
)

// Mistral AI open-weight models.
const (
	MistralOpen7B           = "open-mistral-7b"
	MistralOpenNemo         = "open-mistral-nemo"
	MistralOpenMixtral8x7B  = "open-mixtral-8x7b"
	MistralOpenMixtral8x22B = "open-mixtral-8x22b"
)

// Mistral AI code models.
const (
	MistralCodestralLatest = "codestral-latest"
	MistralCodestral2501   = "codestral-2501"
	MistralCodestral2405   = "codestral-2405"
	MistralDevstralSmall   = "devstral-small-2505"
	MistralDevstralMedium  = "devstral-medium-2507"
)

// Mistral AI compact models.
const (
	MistralMinistral3BLatest = "ministral-3b-latest"
	MistralMinistral8BLatest = "ministral-8b-latest"
)

// Mistral AI vision models.
const (
	MistralPixtralLargeLatest = "pixtral-large-latest"
	MistralPixtral12BLatest   = "pixtral-12b-latest"
	MistralPixtral12B2409     = "pixtral-12b-2409"
)

// Mistral AI embedding models.
const (
	MistralEmbed          = "mistral-embed"
	MistralCodestralEmbed = "codestral-embed"
)

// Backward-compatible aliases.
const (
	MistralLarge     = MistralLargeLatest
	MistralSmall     = MistralSmallLatest
	MistralMedium    = MistralMediumLatest
	MistralNemo      = MistralOpenNemo
	Mistral7B        = MistralOpen7B
	MistralCodestral = MistralCodestralLatest
)
