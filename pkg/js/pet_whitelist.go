package js

// PetScriptWhitelist is used for pet studio project files (DOM + timers allowed).
var PetScriptWhitelist = ASTWhitelist{
	AllowedGlobals: append(DefaultWhitelist.AllowedGlobals,
		"HTMLElement", "HTMLDivElement", "Event", "MouseEvent", "Node", "Element", "PIXI",
	),
	AllowedAPIs: DefaultWhitelist.AllowedAPIs,
	ForbiddenKeywords: []string{
		"eval", "Function", "import", "require",
		"XMLHttpRequest", "fetch", "WebSocket", "Worker",
	},
	ForbiddenFunctions: []string{
		"eval", "Function", "execScript", "document.write", "document.writeln",
	},
	AllowDOMPainting: true,
}
