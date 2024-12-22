package main

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/nlnwa/whatwg-url/canonicalizer"
)

func newDatabase(s *Settings) (neo4j.DriverWithContext, error) {
	uri := "neo4j://localhost:7687" // TODO: add to settings

	driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(s.NEO4J_USER, s.NEO4J_PASSWORD, ""))
	if err != nil {
		return nil, err
	}

	// Test the connection by verifying the authentication
	ctx := context.Background()
	if err := driver.VerifyConnectivity(ctx); err != nil {
		driver.Close(ctx)
		return nil, fmt.Errorf("failed to verify connection: %w", err)
	}

	return driver, nil
}

func PutPage(db neo4j.DriverWithContext, source string, targets []string) error {
	ctx := context.Background()
	session := db.NewSession(ctx, neo4j.SessionConfig{})
	defer session.Close(ctx)

	_, err := neo4j.ExecuteWrite(ctx, session, func(tx neo4j.ManagedTransaction) (any, error) {
		// Step 1: Create the source node
		sourceQuery := "MERGE (source:Page {url: $source})"
		if _, err := tx.Run(ctx, sourceQuery, map[string]any{"source": source}); err != nil {
			return struct{}{}, err
		}

		// Step 2: Create target nodes and relationships
		targetQuery := `
			UNWIND $targets AS targetUrl
			MERGE (target:Page {url: targetUrl})
			MERGE (source:Page {url: $source})-[:LINKS_TO]->(target)
		`
		if _, err := tx.Run(ctx, targetQuery, map[string]any{"source": source, "targets": targets}); err != nil {
			return struct{}{}, err
		}

		// Step 3: Remove edges to nodes not in the targets list
		cleanupQuery := `
			MATCH (source:Page {url: $source})-[r:LINKS_TO]->(target:Page)
			WHERE NOT target.url IN $targets
			DELETE r
		`
		if _, err := tx.Run(ctx, cleanupQuery, map[string]any{"source": source, "targets": targets}); err != nil {
			return struct{}{}, err
		}

		return struct{}{}, nil
	})

	return err
}

func NormalizeUrlString(urlRaw string) (string, error) {
	url, err := canonicalizer.GoogleSafeBrowsing.Parse(urlRaw)
	if err != nil {
		return "", err
	}

	s := url.Scheme()
	if s != "http" && s != "https" {
		return "", fmt.Errorf("url scheme is not http or https: %s", s)
	}

	p := url.Port()
	if p != "" && p != "80" && p != "443" {
		return "", fmt.Errorf("port is not 80 or 443: %s", p)
	}
	url.SetPort("")

	url.SetSearch("")
	url.SetHash("")

	return url.Href(true), nil
}
