package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/LingByte/lingllm/chunk"
)

func main() {
	flag.Parse()
	// Demo 1: Rule-based chunking for structured documents
	demoStructuredChunking()

	// Demo 2: Table/KV chunking
	demoTableKVChunking()

	// Demo 3: Routing chunker (auto-detect document type)
	demoRoutingChunking()

	// Demo 4: Factory pattern usage
	demoFactoryPattern()
}

// demoStructuredChunking demonstrates structured document chunking
func demoStructuredChunking() {
	text := `# Introduction to Go Programming

## Chapter 1: Getting Started

Go is a statically typed, compiled programming language designed for simplicity and efficiency.
It was created by Google in 2009 and has become increasingly popular for building scalable systems.

### Key Features

Go provides several key features that make it attractive for modern software development:
- Fast compilation and execution
- Built-in concurrency support with goroutines
- Simple syntax and small standard library
- Cross-platform compatibility

## Chapter 2: Basic Concepts

### Variables and Types

In Go, variables are declared using the var keyword or the short declaration operator :=.
Go supports various data types including integers, floats, strings, and booleans.

### Functions

Functions in Go are first-class citizens. They can be assigned to variables, passed as arguments,
and returned from other functions. This makes Go a powerful language for functional programming.

### Goroutines

Goroutines are lightweight threads managed by the Go runtime. They allow you to write concurrent
programs that can handle thousands of concurrent operations efficiently.`

	chunker := chunk.NewStructuredRuleChunker(&chunk.Config{
		Provider: "structured",
	})

	opts := &chunk.ChunkOptions{
		MaxChars:      300,
		MinChars:      50,
		OverlapChars:  30,
		DocumentTitle: "Go Programming Guide",
	}

	ctx := context.Background()
	chunks, err := chunker.Chunk(ctx, text, opts)
	if err != nil {
		log.Printf("Error chunking: %v", err)
		return
	}

	fmt.Printf("✓ Chunked into %d segments\n\n", len(chunks))
	for i, c := range chunks {
		fmt.Printf("Chunk %d:\n", i+1)
		fmt.Printf("  Title: %s\n", c.Title)
		fmt.Printf("  Length: %d chars\n", len(c.Text))
		fmt.Printf("  Preview: %s...\n\n", truncate(c.Text, 80))
	}
}

// demoTableKVChunking demonstrates table/KV document chunking
func demoTableKVChunking() {
	text := `Configuration Settings:
name: MyApplication
version: 1.0.0
author: John Doe
email: john@example.com

Database Configuration:
host: localhost
port: 5432
username: admin
password: secret123
database: myapp_db

API Configuration:
base_url: https://api.example.com
timeout: 30
retry_count: 3
rate_limit: 1000

Features:
feature_a: enabled
feature_b: disabled
feature_c: enabled
max_connections: 100`

	chunker := chunk.NewTableKVChunker(&chunk.Config{
		Provider: "table_kv",
	})

	opts := &chunk.ChunkOptions{
		MaxChars:      200,
		MinChars:      30,
		DocumentTitle: "Configuration File",
	}

	ctx := context.Background()
	chunks, err := chunker.Chunk(ctx, text, opts)
	if err != nil {
		log.Printf("Error chunking: %v", err)
		return
	}

	fmt.Printf("✓ Chunked into %d segments\n\n", len(chunks))
	for i, c := range chunks {
		fmt.Printf("Chunk %d:\n", i+1)
		fmt.Printf("  Title: %s\n", c.Title)
		fmt.Printf("  Content:\n%s\n\n", c.Text)
	}
}

// demoRoutingChunking demonstrates automatic document type detection and routing
func demoRoutingChunking() {
	// Example 1: Structured document
	structuredText := `# Machine Learning Basics

## Introduction

Machine learning is a subset of artificial intelligence that enables systems to learn
and improve from experience without being explicitly programmed.

## Types of Machine Learning

### Supervised Learning

Supervised learning involves training a model on labeled data. The model learns to map
input features to output labels.

### Unsupervised Learning

Unsupervised learning works with unlabeled data to discover hidden patterns and structures.`

	// Example 2: Configuration/KV document
	kvText := `server_config:
  host: 0.0.0.0
  port: 8080
  debug: true
  
database:
  type: postgresql
  url: postgres://localhost/mydb
  pool_size: 20`

	chunker := chunk.NewRoutingChunker(&chunk.Config{
		Provider: "router",
	})

	opts := &chunk.ChunkOptions{
		MaxChars:      250,
		MinChars:      40,
		DocumentTitle: "Mixed Content",
	}

	ctx := context.Background()

	fmt.Println("Processing structured document:")
	chunks1, err := chunker.Chunk(ctx, structuredText, opts)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("✓ Detected as STRUCTURED, chunked into %d segments\n\n", len(chunks1))

	fmt.Println("Processing configuration document:")
	chunks2, err := chunker.Chunk(ctx, kvText, opts)
	if err != nil {
		log.Printf("Error: %v", err)
		return
	}
	fmt.Printf("✓ Detected as TABLE_KV, chunked into %d segments\n\n", len(chunks2))
}

// demoFactoryPattern demonstrates the factory pattern for chunker creation
func demoFactoryPattern() {
	text := `# Quick Start Guide

## Installation

To get started, install the package using your package manager.

## Configuration

Configure the system by setting the following parameters:
timeout: 30
retries: 3
debug: false

## Usage

Here's a simple example of how to use the system.`

	// Get the global factory
	factory := chunk.GetFactory()

	fmt.Println("Available chunkers:")
	for _, provider := range factory.List() {
		fmt.Printf("  - %s\n", provider)
	}
	fmt.Println()

	// Create chunkers using factory
	providers := []string{"rules_structured", "rules_table_kv", "router"}

	ctx := context.Background()

	for _, provider := range providers {
		fmt.Printf("Creating %s chunker...\n", provider)

		cfg := &chunk.Config{
			Provider: provider,
		}

		chunker, err := factory.Create(ctx, cfg)
		if err != nil {
			log.Printf("Error creating chunker: %v", err)
			continue
		}

		opts := &chunk.ChunkOptions{
			MaxChars: 200,
			MinChars: 30,
		}

		chunks, err := chunker.Chunk(ctx, text, opts)
		if err != nil {
			log.Printf("Error chunking: %v", err)
			continue
		}

		fmt.Printf("✓ %s chunker produced %d chunks\n\n", provider, len(chunks))
	}
}

// Helper function to truncate text
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
