package main

import (
	"fmt"
	"log"

	"github.com/LingByte/lingllm/prompt"
)

func main() {
	fmt.Println("=== Prompt Template Demo ===")

	// 1. Basic Template Usage
	fmt.Println("1. Basic Template Usage")
	basicTemplate()

	// 2. Variable Substitution
	fmt.Println("\n2. Variable Substitution")
	variableSubstitution()

	// 3. Nested Variables and Arrays
	fmt.Println("\n3. Nested Variables and Arrays")
	nestedVariables()

	// 4. Template Set
	fmt.Println("\n4. Template Set")
	templateSet()

	// 5. Message Templates
	fmt.Println("\n5. Message Templates")
	messageTemplates()

	// 6. Few-Shot Learning
	fmt.Println("\n6. Few-Shot Learning")
	fewShotLearning()

	// 7. Chain of Thought
	fmt.Println("\n7. Chain of Thought")
	chainOfThought()

	// 8. Global Registry
	fmt.Println("\n8. Global Registry")
	globalRegistry()
}

func basicTemplate() {
	tpl, err := prompt.NewTemplate("greeting", "Hello, {{name}}!")
	if err != nil {
		log.Fatal(err)
	}

	result, err := tpl.Render(map[string]any{"name": "World"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	// Output: Hello, World!
}

func variableSubstitution() {
	tpl := prompt.MustNewTemplate("user", "User: {{user.name}}, Age: {{user.age}}")

	result, err := tpl.Render(map[string]any{
		"user": map[string]any{
			"name": "Alice",
			"age":  30,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	// Output: User: Alice, Age: 30

	// Default values
	tpl2 := prompt.MustNewTemplate("greeting", "Hello, {{name|Guest}}!")
	result2, err := tpl2.Render(map[string]any{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result2)
	// Output: Hello, Guest!
}

func nestedVariables() {
	// Array access
	tpl := prompt.MustNewTemplate("items", "First: {{items[0]}}, Second: {{items[1]}}")
	result, err := tpl.Render(map[string]any{
		"items": []any{"apple", "banana", "cherry"},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
	// Output: First: apple, Second: banana

	// Nested array access
	tpl2 := prompt.MustNewTemplate("nested", "User: {{users[0].name}}")
	result2, err := tpl2.Render(map[string]any{
		"users": []any{
			map[string]any{"name": "Bob", "age": 25},
			map[string]any{"name": "Carol", "age": 28},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result2)
	// Output: User: Bob
}

func templateSet() {
	// Create a template set
	ts := prompt.NewTemplateSet()

	// Register templates
	ts.ParseAndRegister("greeting", "Hello, {{name}}!")
	ts.ParseAndRegister("farewell", "Goodbye, {{name}}!")
	ts.ParseAndRegister("welcome", "Welcome to {{company}}, {{name}}!")

	// Add default functions
	ts.AddDefaultFuncs()

	// Render templates
	result1, err := ts.Render("greeting", map[string]any{"name": "Alice"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result1)

	result2, err := ts.Render("welcome", map[string]any{"name": "Bob", "company": "Acme Inc"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result2)

	// List all templates
	fmt.Println("Registered templates:", ts.List())

	// Extend a template
	ts.ParseAndRegister("base", "Base template with {{content}}")
	ts.Extend("extended", "base", "{{$super}} - Extended version")

	result3, err := ts.Render("extended", map[string]any{"content": "data"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result3)
}

func messageTemplates() {
	// Create a conversation
	conv := prompt.NewConversation("assistant")

	// Add system message
	conv.AddSystem(prompt.MustNewTemplate("sys", "You are a helpful {{role}} assistant."))

	// Add user message with template
	conv.AddUser(prompt.MustNewTemplate("user", "What is {{topic}}?"))

	// Add conditional message
	adminMsg := prompt.NewMessageTemplate(prompt.RoleSystem, prompt.MustNewTemplate("admin", "Admin Panel Access"))
	adminMsg.WithCondition("user.isAdmin", "eq", true)
	conv.AddConditional(adminMsg)

	// Render conversation
	data := map[string]any{
		"role":  "coding",
		"topic": "Go",
		"user":  map[string]any{"isAdmin": false},
	}

	messages, err := conv.Render(data)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Messages:")
	for _, msg := range messages {
		fmt.Printf("  [%s] %s\n", msg.Role, msg.Content)
	}
}

func fewShotLearning() {
	// Create a few-shot template
	fst := prompt.NewFewShotTemplate()

	// Add examples
	fst.AddExample(map[string]any{"input": "Hello"}, "Hi there!")
	fst.AddExample(map[string]any{"input": "How are you?"}, "I'm doing great, thanks!")
	fst.AddExample(map[string]any{"input": "Goodbye"}, "See you later!")

	// Render examples
	result, err := fst.Render()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Default Few-Shot Format:")
	fmt.Println(result)

	// Custom format
	fst2 := prompt.NewFewShotTemplate()
	fst2.WithExampleTemplate(prompt.MustNewTemplate("example", "Q: {{input}}\nA: {{output}}"))
	fst2.AddExample(map[string]any{"input": "2 + 2 = ?"}, "4")
	fst2.AddExample(map[string]any{"input": "3 + 3 = ?"}, "6")
	fst2.WithSeparator("\n---\n")

	result2, err := fst2.Render()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nCustom Few-Shot Format:")
	fmt.Println(result2)

	// Example Selector
	selector := prompt.NewExampleSelector()
	selector.AddExamples(
		prompt.Example{Input: map[string]any{"q": "What is Go?"}, Output: "Go is a programming language."},
		prompt.Example{Input: map[string]any{"q": "What is Python?"}, Output: "Python is a programming language."},
	)
	selector.WithLimit(1)

	examples, err := selector.Select(map[string]any{})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\nSelected Examples:", len(examples))
}

func chainOfThought() {
	// Create a reasoning template
	rt := prompt.NewReasoningTemplate()

	// Use XML format
	rt.WithFormat(prompt.FormatXML)

	// Build system prompt
	systemPrompt := rt.BuildSystemPrompt()
	fmt.Println("CoT System Prompt (XML format):")
	fmt.Println(systemPrompt)

	// Parse steps from response
	response := `<thought>I need to solve 2+2</thought>
<action>Add the numbers</action>
<result>The answer is 4</result>`

	steps, err := rt.ParseSteps(response)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\nParsed Steps:")
	for i, step := range steps {
		fmt.Printf("  Step %d:\n", i+1)
		fmt.Printf("    Thought: %s\n", step.Thought)
		fmt.Printf("    Action: %s\n", step.Action)
		fmt.Printf("    Result: %s\n", step.Result)
	}

	// Convert to conversation
	conv := rt.ToConversation("What is 2+2?")
	messages, _ := conv.Render(map[string]any{})
	fmt.Println("\nConversation created with", len(messages), "messages")
}

func globalRegistry() {
	// Clear the global registry
	prompt.TemplateRegistry = prompt.NewTemplateSet()

	// Register globally
	prompt.ParseAndRegisterGlobal("hello", "Hello, {{name}}!")
	prompt.ParseAndRegisterGlobal("thanks", "Thank you, {{name}}, for {{action}}!")

	// Render globally
	result1, err := prompt.RenderGlobal("hello", map[string]any{"name": "World"})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result1)

	result2, err := prompt.RenderGlobal("thanks", map[string]any{
		"name":   "Alice",
		"action": "using this app",
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result2)
}
