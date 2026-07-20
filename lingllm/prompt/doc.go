// Package prompt provides comprehensive prompt template management for LLM applications.
//
// # Core Features
//
//   - Variable substitution: {{variable}}, {{nested.key}}
//   - Conditional logic: {{#if condition}}...{{/if}}, {{#else}}...{{/else}}
//   - Loop iteration: {{#each items}}...{{/each}}, {{@index}}, {{@key}}
//   - Template composition: include, extend, partials
//   - Message templates: system, user, assistant
//   - Few-shot examples with demonstrations
//   - Chain of thought (CoT) prompting
//   - Tree of thought (ToT) reasoning
//   - Self-reflection prompting
//
// # Quick Start
//
//	renderer, err := prompt.NewTemplate("hello", "Hello, {{name}}!")
//	result, err := renderer.Render(map[string]any{"name": "World"})
//	// result = "Hello, World!"
//
// # Variable Substitution
//
// Templates support dot notation for nested values:
//
//	"User: {{user.name}}, Age: {{user.profile.age}}"
//
// Array access is also supported:
//
//	"First item: {{items[0]}}"
//
// Default values:
//
//	"Hello, {{name|Anonymous}}!"
//
// # Conditionals
//
//	"{{#if user.isAdmin}}Admin Panel{{/if}}"
//	"{{#if count > 0}}{{count}} items{{#else}}No items{{/if}}"
//
// # Loops
//
//	"{{#each items as item}}[{{@index}}] {{item}}{{/each}}"
//	"{{#each users as user key idx}}User {{key}}: {{user.name}}{{/each}}"
//
// # Template Sets
//
// Organize and reuse templates with a TemplateSet:
//
//	ts := prompt.NewTemplateSet()
//	ts.ParseAndRegister("greeting", "Hello, {{name}}!")
//	ts.ParseAndRegister("farewell", "Goodbye, {{name}}!")
//
//	result, err := ts.Render("greeting", map[string]any{"name": "Alice"})
//
// # Few-Shot Learning
//
//	fst := prompt.NewFewShotTemplate()
//	fst.AddExample(map[string]any{"input": "Hello"}, "Hi there!")
//	fst.AddExample(map[string]any{"input": "Goodbye"}, "See you later!")
//	fewShotContent, _ := fst.Render()
//
// # Chain of Thought
//
//	rt := prompt.NewReasoningTemplate()
//	rt.WithFormat(prompt.FormatXML)
//	systemPrompt := rt.BuildSystemPrompt()
//
// # Message Templates
//
// Build structured conversations:
//
//	conv := prompt.NewConversation("assistant")
//	conv.AddSystem(prompt.MustNewTemplate("sys", "You are a helpful assistant."))
//	conv.AddUser(prompt.MustNewTemplate("user", "{{question}}"))
//
//	messages, _ := conv.Render(map[string]any{"question": "What is Go?"})
//
// # Global Registry
//
// Use the global registry for convenience:
//
//	prompt.ParseAndRegisterGlobal("hello", "Hello, {{name}}!")
//	result, _ := prompt.RenderGlobal("hello", map[string]any{"name": "World"})
package prompt
