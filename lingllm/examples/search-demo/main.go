package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/LingByte/lingllm/search"
)

func main() {
	indexPath := flag.String("index", filepath.Join(os.TempDir(), "search_demo_index"), "Index path")
	flag.Parse()
	// Create search engine
	cfg := search.Config{
		IndexPath:           *indexPath,
		DefaultAnalyzer:     "standard",
		DefaultSearchFields: []string{"title", "body", "tags"},
		QueryTimeout:        10 * time.Second,
		BatchSize:           100,
	}

	m := search.BuildIndexMapping("standard")
	engine, err := search.New(cfg, m)
	if err != nil {
		log.Fatalf("Failed to create search engine: %v", err)
	}
	defer engine.Close()

	fmt.Println("✓ Search engine initialized")
	fmt.Printf("  Index path: %s\n\n", *indexPath)

	// Demo 1: Index documents
	demoIndexDocuments(engine)

	// Demo 2: Basic search
	demoBasicSearch(engine)

	// Demo 3: Advanced search
	demoAdvancedSearch(engine)

	// Demo 4: Faceted search
	demoFacetedSearch(engine)

	// Demo 5: Suggestions
	demoSuggestions(engine)

	// Demo 6: Highlighting
	demoHighlighting(engine)

	// Cleanup
	os.RemoveAll(*indexPath)
}

func demoIndexDocuments(engine search.Engine) {

	docs := []search.Doc{
		{
			ID:   "1",
			Type: "article",
			Fields: map[string]interface{}{
				"title":    "Introduction to Go Programming",
				"body":     "Go is a statically typed, compiled programming language designed for simplicity and efficiency.",
				"tags":     "golang,programming,tutorial",
				"author":   "Alice",
				"category": "programming",
				"views":    1500,
			},
		},
		{
			ID:   "2",
			Type: "article",
			Fields: map[string]interface{}{
				"title":    "Python for Data Science",
				"body":     "Python has become the de facto language for data science and machine learning applications.",
				"tags":     "python,data-science,machine-learning",
				"author":   "Bob",
				"category": "data-science",
				"views":    2300,
			},
		},
		{
			ID:   "3",
			Type: "article",
			Fields: map[string]interface{}{
				"title":    "Web Development with Go",
				"body":     "Building web applications with Go is fast, efficient, and scalable.",
				"tags":     "golang,web-development,backend",
				"author":   "Charlie",
				"category": "programming",
				"views":    1800,
			},
		},
		{
			ID:   "4",
			Type: "article",
			Fields: map[string]interface{}{
				"title":    "Machine Learning Basics",
				"body":     "Machine learning enables computers to learn from data without being explicitly programmed.",
				"tags":     "machine-learning,ai,algorithms",
				"author":   "Diana",
				"category": "data-science",
				"views":    2100,
			},
		},
		{
			ID:   "5",
			Type: "article",
			Fields: map[string]interface{}{
				"title":    "Rust Systems Programming",
				"body":     "Rust provides memory safety without garbage collection, making it ideal for systems programming.",
				"tags":     "rust,systems,programming",
				"author":   "Eve",
				"category": "programming",
				"views":    1200,
			},
		},
	}

	err := engine.IndexBatch(context.Background(), docs)
	if err != nil {
		log.Fatalf("Failed to index documents: %v", err)
	}

	fmt.Printf("✓ Indexed %d documents\n\n", len(docs))
	for _, doc := range docs {
		fmt.Printf("  - %s: %s\n", doc.ID, doc.Fields["title"])
	}
	fmt.Println()
}

func demoBasicSearch(engine search.Engine) {

	queries := []string{"Go", "Python", "machine learning"}

	for _, q := range queries {
		result, err := engine.Search(context.Background(), search.SearchRequest{
			Keyword: q,
			Size:    10,
		})
		if err != nil {
			log.Printf("Search failed: %v", err)
			continue
		}

		fmt.Printf("Query: \"%s\"\n", q)
		fmt.Printf("Results: %d hits (took %v)\n", result.Total, result.Took)
		for i, hit := range result.Hits {
			fmt.Printf("  %d. [%.2f] %s\n", i+1, hit.Score, hit.Fields["title"])
		}
		fmt.Println()
	}
}

func demoAdvancedSearch(engine search.Engine) {

	// Search with must terms
	fmt.Println("Search: Programming articles with high views")
	result, err := engine.Search(context.Background(), search.SearchRequest{
		MustTerms: map[string][]string{
			"category": {"programming"},
		},
		Matches: []search.ClauseMatch{
			{
				Field: "title",
				Query: "programming",
			},
		},
		Size: 10,
	})
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Results: %d hits\n", result.Total)
		for i, hit := range result.Hits {
			fmt.Printf("  %d. [%.2f] %s (Category: %s)\n", i+1, hit.Score, hit.Fields["title"], hit.Fields["category"])
		}
	}
	fmt.Println()
}

func demoFacetedSearch(engine search.Engine) {

	result, err := engine.Search(context.Background(), search.SearchRequest{
		Keyword: "programming",
		Facets: []search.FacetRequest{
			{
				Name:  "categories",
				Field: "category",
				Size:  10,
			},
		},
		Size: 10,
	})
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	fmt.Printf("Query: \"programming\"\n")
	fmt.Printf("Results: %d hits\n\n", result.Total)

	if facets, ok := result.Facets["categories"]; ok {
		fmt.Println("Categories:")
		for _, term := range facets.Terms {
			fmt.Printf("  - %s: %d\n", term.Term, term.Count)
		}
	}
	fmt.Println()
}

func demoSuggestions(engine search.Engine) {

	// Autocomplete suggestions
	keywords := []string{"Go", "Py", "Mach"}

	for _, kw := range keywords {
		suggestions, err := engine.GetAutoCompleteSuggestions(context.Background(), kw)
		if err != nil {
			log.Printf("Failed to get suggestions: %v", err)
			continue
		}

		fmt.Printf("Autocomplete for \"%s\":\n", kw)
		if len(suggestions) > 0 {
			for i, s := range suggestions {
				fmt.Printf("  %d. %s\n", i+1, s)
			}
		} else {
			fmt.Println("  (no suggestions)")
		}
		fmt.Println()
	}
}

func demoHighlighting(engine search.Engine) {

	result, err := engine.Search(context.Background(), search.SearchRequest{
		Keyword:         "programming",
		Highlight:       true,
		HighlightFields: []string{"title", "body"},
		Size:            5,
	})
	if err != nil {
		log.Printf("Search failed: %v", err)
		return
	}

	fmt.Printf("Query: \"programming\" (with highlighting)\n")
	fmt.Printf("Results: %d hits\n\n", result.Total)

	for i, hit := range result.Hits {
		fmt.Printf("%d. %s\n", i+1, hit.Fields["title"])
		if len(hit.Fragments) > 0 {
			fmt.Println("   Highlights:")
			for field, frags := range hit.Fragments {
				fmt.Printf("   - %s:\n", field)
				for _, frag := range frags {
					fmt.Printf("     %s\n", frag)
				}
			}
		}
		fmt.Println()
	}
}
