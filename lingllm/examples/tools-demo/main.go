package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/LingByte/lingllm/examples/exutil"
	"github.com/LingByte/lingllm/protocol"
	_ "github.com/LingByte/lingllm/protocol/openai"
	"github.com/LingByte/lingllm/tools"
)

func main() {
	apiKey := flag.String("apikey", "", "API key for the LLM provider")
	model := flag.String("model", "gpt-4", "Model name")
	baseURL := flag.String("base_url", "", "Base URL for the API")
	flag.Parse()

	if *apiKey == "" {
		log.Fatal("apikey is required")
	}

	// Create the LLM client
	client, err := protocol.NewClient(protocol.ClientConfig{
		Provider: "openai",
		APIKey:   *apiKey,
		BaseURL:  *baseURL,
	})
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Create tool executor
	executor := tools.NewSimpleToolExecutor()

	// Register tools
	registerTools(executor)

	// Create tool chain
	toolChain := tools.NewToolChain(client, executor)
	toolChain.WithMaxRounds(5)

	ctx := context.Background()

	fmt.Println("=== LingLLM Tools Demo ===")

	// Test 1: Simple calculation
	fmt.Println("Test 1: Simple Calculation")
	fmt.Println("─────────────────────────────────────")
	testCalculation(ctx, toolChain, *model)

	// Test 2: Weather lookup
	fmt.Println("\nTest 2: Weather Lookup")
	fmt.Println("─────────────────────────────────────")
	testWeather(ctx, toolChain, *model)

	// Test 3: Web search
	fmt.Println("\nTest 3: Web Search")
	fmt.Println("─────────────────────────────────────")
	testSearch(ctx, toolChain, *model)

	// Test 4: Multi-step tool usage
	fmt.Println("\nTest 4: Multi-step Tool Usage")
	fmt.Println("─────────────────────────────────────")
	testMultiStep(ctx, toolChain, *model)
}

func registerTools(executor *tools.SimpleToolExecutor) {
	// Register calculator tool
	executor.RegisterTool(tools.CalculatorTool(), func(args json.RawMessage) (string, error) {
		var input struct {
			Expression string `json:"expression"`
		}
		if err := json.Unmarshal(args, &input); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		return evaluateExpression(input.Expression)
	})

	// Register weather tool
	executor.RegisterTool(tools.WeatherTool(), func(args json.RawMessage) (string, error) {
		var input struct {
			Location string `json:"location"`
			Unit     string `json:"unit"`
		}
		if err := json.Unmarshal(args, &input); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		return getWeather(input.Location, input.Unit)
	})

	// Register web search tool
	executor.RegisterTool(tools.SearchTool(), func(args json.RawMessage) (string, error) {
		var input struct {
			Query      string `json:"query"`
			MaxResults int    `json:"max_results"`
		}
		if err := json.Unmarshal(args, &input); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		if input.MaxResults == 0 {
			input.MaxResults = 10
		}
		return webSearch(input.Query, input.MaxResults)
	})
}

func testCalculation(ctx context.Context, tc *tools.ToolChain, model string) {
	req := protocol.ChatRequest{
		Model: model,
		Messages: []protocol.Message{
			{
				Role:    protocol.RoleUser,
				Content: "What is 25 * 4 + 10 / 2? Show me the calculation step by step.",
			},
		},
		MaxTokens: 500,
	}

	e2eStart := time.Now()
	resp, err := tc.ExecuteWithTools(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	exutil.LogChat("calculation", resp, e2eStart)

	fmt.Printf("Question: What is 25 * 4 + 10 / 2?\n")
	fmt.Printf("Answer: %s\n", resp.FirstContent())
}

func testWeather(ctx context.Context, tc *tools.ToolChain, model string) {
	req := protocol.ChatRequest{
		Model: model,
		Messages: []protocol.Message{
			{
				Role:    protocol.RoleUser,
				Content: "What's the weather like in San Francisco, CA in Celsius?",
			},
		},
		MaxTokens: 500,
	}

	e2eStart := time.Now()
	resp, err := tc.ExecuteWithTools(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	exutil.LogChat("weather", resp, e2eStart)

	fmt.Printf("Question: What's the weather like in San Francisco, CA in Celsius?\n")
	fmt.Printf("Answer: %s\n", resp.FirstContent())
}

func testSearch(ctx context.Context, tc *tools.ToolChain, model string) {
	req := protocol.ChatRequest{
		Model: model,
		Messages: []protocol.Message{
			{
				Role:    protocol.RoleUser,
				Content: "Search for information about Go programming language and summarize the top 3 results.",
			},
		},
		MaxTokens: 500,
	}

	e2eStart := time.Now()
	resp, err := tc.ExecuteWithTools(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	exutil.LogChat("search", resp, e2eStart)

	fmt.Printf("Question: Search for information about Go programming language\n")
	fmt.Printf("Answer: %s\n", resp.FirstContent())
}

func testMultiStep(ctx context.Context, tc *tools.ToolChain, model string) {
	req := protocol.ChatRequest{
		Model: model,
		Messages: []protocol.Message{
			{
				Role:    protocol.RoleUser,
				Content: "I need to calculate the area of a circle with radius 5. First, calculate 5 * 5 * 3.14159, then tell me what the result means.",
			},
		},
		MaxTokens: 500,
	}

	e2eStart := time.Now()
	resp, err := tc.ExecuteWithTools(ctx, req)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	exutil.LogChat("multi-step", resp, e2eStart)

	fmt.Printf("Question: Calculate area of circle with radius 5\n")
	fmt.Printf("Answer: %s\n", resp.FirstContent())
}

// Mock implementations of tools
func evaluateExpression(expr string) (string, error) {
	// Simple expression evaluator
	expr = strings.TrimSpace(expr)

	// Handle basic operations
	if strings.Contains(expr, "+") {
		parts := strings.Split(expr, "+")
		if len(parts) == 2 {
			a, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			b, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			result := a + b
			return fmt.Sprintf("%.2f", result), nil
		}
	}

	if strings.Contains(expr, "-") {
		parts := strings.Split(expr, "-")
		if len(parts) == 2 {
			a, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			b, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			result := a - b
			return fmt.Sprintf("%.2f", result), nil
		}
	}

	if strings.Contains(expr, "*") {
		parts := strings.Split(expr, "*")
		if len(parts) == 2 {
			a, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			b, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			result := a * b
			return fmt.Sprintf("%.2f", result), nil
		}
	}

	if strings.Contains(expr, "/") {
		parts := strings.Split(expr, "/")
		if len(parts) == 2 {
			a, _ := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			b, _ := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if b == 0 {
				return "", fmt.Errorf("division by zero")
			}
			result := a / b
			return fmt.Sprintf("%.2f", result), nil
		}
	}

	// Complex expression with multiple operations
	// For demo: 25 * 4 + 10 / 2 = 100 + 5 = 105
	if expr == "25 * 4 + 10 / 2" {
		return "105.00", nil
	}

	// For demo: 5 * 5 * 3.14159 = 78.53975
	if expr == "5 * 5 * 3.14159" {
		return fmt.Sprintf("%.5f", 5*5*3.14159), nil
	}

	return "", fmt.Errorf("unsupported expression: %s", expr)
}

func getWeather(location string, unit string) (string, error) {
	// Mock weather data
	weatherData := map[string]map[string]string{
		"San Francisco, CA": {
			"celsius":    "15°C, Partly Cloudy",
			"fahrenheit": "59°F, Partly Cloudy",
		},
		"New York, NY": {
			"celsius":    "22°C, Sunny",
			"fahrenheit": "72°F, Sunny",
		},
		"London, UK": {
			"celsius":    "12°C, Rainy",
			"fahrenheit": "54°F, Rainy",
		},
	}

	if unit == "" {
		unit = "celsius"
	}

	if weather, ok := weatherData[location]; ok {
		if temp, ok := weather[unit]; ok {
			return fmt.Sprintf("Weather in %s: %s", location, temp), nil
		}
	}

	return fmt.Sprintf("Weather in %s: 20°C, Clear skies (mock data)", location), nil
}

func webSearch(query string, maxResults int) (string, error) {
	// Mock search results
	searchResults := map[string][]string{
		"Go programming language": {
			"1. Go is an open-source programming language created by Google in 2007",
			"2. Go is known for its simplicity, efficiency, and strong support for concurrent programming",
			"3. Go is widely used for building web services, cloud infrastructure, and DevOps tools",
		},
		"Python": {
			"1. Python is a high-level, interpreted programming language",
			"2. Python is known for its simplicity and readability",
			"3. Python is widely used in data science, machine learning, and web development",
		},
		"Rust": {
			"1. Rust is a systems programming language focused on safety and performance",
			"2. Rust provides memory safety without garbage collection",
			"3. Rust is used for building high-performance and reliable software",
		},
	}

	results := searchResults[query]
	if len(results) == 0 {
		results = []string{
			fmt.Sprintf("1. Search result for '%s' - No specific data available", query),
			"2. This is a mock search result for demonstration purposes",
			"3. In a real implementation, this would call an actual search API",
		}
	}

	if maxResults > len(results) {
		maxResults = len(results)
	}

	return strings.Join(results[:maxResults], "\n"), nil
}
