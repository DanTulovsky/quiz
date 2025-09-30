// Package main is a script to check for undocumented APIs in the router.
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
)

// Endpoint represents an API endpoint with its method and path.
type Endpoint struct {
	Method string
	Path   string
}

func main() {
	// Get router endpoints by analyzing the router file
	routerEndpoints := extractRouterEndpoints()

	// Get swagger endpoints
	swaggerEndpoints := extractSwaggerEndpoints()

	// Find missing endpoints
	missingEndpoints := findMissingEndpoints(routerEndpoints, swaggerEndpoints)

	if len(missingEndpoints) == 0 {
		fmt.Println("✅ All router endpoints are documented in swagger.yaml")
		return
	}

	// Show only the missing endpoints
	fmt.Println("❌ Router endpoints not in swagger:")
	for _, endpoint := range missingEndpoints {
		fmt.Printf("  %s %s\n", endpoint.Method, endpoint.Path)
	}

	fmt.Println("\n⚠️  Please add missing endpoints to swagger.yaml")
	fmt.Println("Run 'task generate-api-types' after adding missing endpoints")

	// Exit with error code to indicate failure
	os.Exit(1)
}

func extractRouterEndpoints() []Endpoint {
	routerFile := "backend/internal/handlers/router_factory.go"
	content, err := ioutil.ReadFile(routerFile)
	if err != nil {
		log.Printf("Error reading router file: %v", err)
		return nil
	}

	var endpoints []Endpoint

	// Extract endpoints using regex patterns
	patterns := []struct {
		pattern *regexp.Regexp
		group   string
	}{
		// Router endpoints
		{regexp.MustCompile(`router\.(GET|POST|PUT|DELETE|PATCH)\("([^"]+)"`), ""},
		// Auth group endpoints
		{regexp.MustCompile(`auth\.(GET|POST|PUT|DELETE|PATCH)\("([^"]+)"`), "/v1/auth"},
		// Quiz group endpoints
		{regexp.MustCompile(`quiz\.(GET|POST|PUT|DELETE|PATCH)\("([^"]+)"`), "/v1/quiz"},
		// Settings group endpoints
		{regexp.MustCompile(`settings\.(GET|POST|PUT|DELETE|PATCH)\("([^"]+)"`), "/v1/settings"},
		// Preferences group endpoints
		{regexp.MustCompile(`preferences\.(GET|POST|PUT|DELETE|PATCH)\("([^"]+)"`), "/v1/preferences"},
		// Userz group endpoints
		{regexp.MustCompile(`userz\.(GET|POST|PUT|DELETE|PATCH)\("([^"]+)"`), "/v1/userz"},
		// Backend group endpoints
		{regexp.MustCompile(`backend\.(GET|POST|PUT|DELETE|PATCH)\("([^"]+)"`), "/v1/admin/backend"},
	}

	for _, p := range patterns {
		matches := p.pattern.FindAllStringSubmatch(string(content), -1)
		for _, match := range matches {
			if len(match) >= 3 {
				method := match[1]
				path := match[2]

				// Skip function definitions and other non-endpoint matches
				if strings.Contains(path, "func") || strings.Contains(path, "http.NewRequest") {
					continue
				}

				// Construct full path
				fullPath := path
				if p.group != "" {
					fullPath = p.group + path
				} else if !strings.HasPrefix(path, "/v1/") && !strings.HasPrefix(path, "/health") && !strings.HasPrefix(path, "/configz") && !strings.HasPrefix(path, "/") {
					fullPath = "/v1" + path
				}

				endpoints = append(endpoints, Endpoint{
					Method: method,
					Path:   fullPath,
				})
			}
		}
	}

	// Remove duplicates and sort
	seen := make(map[string]bool)
	var uniqueEndpoints []Endpoint
	for _, endpoint := range endpoints {
		key := endpoint.Method + " " + endpoint.Path
		if !seen[key] {
			seen[key] = true
			uniqueEndpoints = append(uniqueEndpoints, endpoint)
		}
	}

	sort.Slice(uniqueEndpoints, func(i, j int) bool {
		if uniqueEndpoints[i].Path != uniqueEndpoints[j].Path {
			return uniqueEndpoints[i].Path < uniqueEndpoints[j].Path
		}
		return uniqueEndpoints[i].Method < uniqueEndpoints[j].Method
	})

	return uniqueEndpoints
}

func extractSwaggerEndpoints() []Endpoint {
	swaggerFile := "swagger.yaml"
	content, err := ioutil.ReadFile(swaggerFile)
	if err != nil {
		log.Printf("Error reading swagger file: %v", err)
		return nil
	}

	var endpoints []Endpoint
	lines := strings.Split(string(content), "\n")

	// Simple regex to extract paths and methods from swagger.yaml
	pathRegex := regexp.MustCompile(`^  (/[^:]*):$`)
	methodRegex := regexp.MustCompile(`^    (get|post|put|delete|patch):$`)

	var currentPath string
	for _, line := range lines {
		// Check for path definition
		if pathMatch := pathRegex.FindStringSubmatch(line); len(pathMatch) > 1 {
			currentPath = pathMatch[1]
			continue
		}

		// Check for method definition
		if methodMatch := methodRegex.FindStringSubmatch(line); len(methodMatch) > 1 && currentPath != "" {
			method := strings.ToUpper(methodMatch[1])
			endpoints = append(endpoints, Endpoint{
				Method: method,
				Path:   currentPath,
			})
		}
	}

	// Sort endpoints
	sort.Slice(endpoints, func(i, j int) bool {
		if endpoints[i].Path != endpoints[j].Path {
			return endpoints[i].Path < endpoints[j].Path
		}
		return endpoints[i].Method < endpoints[j].Method
	})

	return endpoints
}

func findMissingEndpoints(routerEndpoints, swaggerEndpoints []Endpoint) []Endpoint {
	swaggerMap := make(map[string]bool)
	for _, endpoint := range swaggerEndpoints {
		key := endpoint.Method + " " + endpoint.Path
		swaggerMap[key] = true
	}

	// Define endpoints to exclude from the check
	excludedEndpoints := map[string]bool{
		"GET /configz": true,
	}

	var missing []Endpoint
	for _, endpoint := range routerEndpoints {
		key := endpoint.Method + " " + endpoint.Path

		// Also check with path parameter format conversion
		convertedPath := convertPathParams(endpoint.Path)
		convertedKey := endpoint.Method + " " + convertedPath

		// Skip the root endpoint as it's just a route listing
		if endpoint.Path == "/v1/" || endpoint.Path == "/" {
			continue
		}

		// Skip excluded endpoints
		if excludedEndpoints[key] {
			continue
		}

		if !swaggerMap[key] && !swaggerMap[convertedKey] {
			missing = append(missing, endpoint)
		}
	}

	return missing
}

func convertPathParams(path string) string {
	// Convert router path parameters (:id) to swagger format ({id})
	re := regexp.MustCompile(`:([^/]+)`)
	return re.ReplaceAllString(path, "{$1}")
}
