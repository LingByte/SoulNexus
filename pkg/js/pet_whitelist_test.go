package js

import "testing"

func TestPetScriptWhitelistAllowsInnerHTML(t *testing.T) {
	code := `root.innerHTML = '<div id="app"></div>'`
	valid, violations := ValidateAST(code, PetScriptWhitelist)
	if !valid {
		t.Fatalf("expected innerHTML allowed for pet scripts, got: %v", violations)
	}
}

func TestPetScriptWhitelistAllowsSetTimeout(t *testing.T) {
	code := `(function() {
  setTimeout(function() {}, 150)
  document.addEventListener('click', function() {})
})()`
	valid, violations := ValidateAST(code, PetScriptWhitelist)
	if !valid {
		t.Fatalf("expected valid pet script, got violations: %v", violations)
	}
}

func TestDefaultWhitelistBlocksSetTimeout(t *testing.T) {
	code := `setTimeout(function() {}, 150)`
	valid, _ := ValidateAST(code, DefaultWhitelist)
	if valid {
		t.Fatal("DefaultWhitelist should block setTimeout")
	}
}
