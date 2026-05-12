package codescan

import (
	"regexp"
	"strings"
)

// ExtractEndpoints detects API endpoints from source code content.
// Supports: Go (net/http, gin, echo, chi, mux, fiber), JS/TS (Express, Fastify, Koa, Hono, Next.js),
// Python (Flask, FastAPI, Django), Java (Spring, JAX-RS), Groovy (Grails), gRPC (.proto),
// and GraphQL schemas.
func ExtractEndpoints(path string, content []byte, lang string) []Endpoint {
	text := string(content)
	lines := strings.Split(text, "\n")

	var endpoints []Endpoint

	switch lang {
	case "go":
		endpoints = append(endpoints, extractGoEndpoints(path, lines)...)
	case "typescript", "javascript":
		endpoints = append(endpoints, extractJSTSEndpoints(path, lines)...)
	case "python":
		endpoints = append(endpoints, extractPythonEndpoints(path, lines)...)
	case "java":
		endpoints = append(endpoints, extractJavaEndpoints(path, lines)...)
	case "groovy":
		endpoints = append(endpoints, extractGroovyEndpoints(path, lines)...)
	}

	// Protocol-specific file types
	if strings.HasSuffix(path, ".proto") {
		endpoints = append(endpoints, extractProtoEndpoints(path, lines)...)
	}
	if strings.HasSuffix(path, ".graphql") || strings.HasSuffix(path, ".gql") {
		endpoints = append(endpoints, extractGraphQLEndpoints(path, lines)...)
	}

	return endpoints
}

// --- Go endpoint patterns ---

var (
	// http.HandleFunc("/path", handler)
	goHandleFuncRe = regexp.MustCompile(`(?:HandleFunc|Handle)\s*\(\s*"([^"]+)"`)
	// r.GET("/path", handler) — gin, echo, fiber, chi
	goRouterMethodRe = regexp.MustCompile(`\.\s*(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\s*\(\s*"([^"]+)"(?:.*,\s*(\w+))?`)
	// r.HandleFunc("/path", handler).Methods("GET")
	goMuxRe = regexp.MustCompile(`HandleFunc\s*\(\s*"([^"]+)".*\.Methods\s*\(\s*"(\w+)"`)
	// router.Group("/api")
	goGroupRe = regexp.MustCompile(`\.(?:Group|PathPrefix)\s*\(\s*"([^"]+)"`)
)

func extractGoEndpoints(path string, lines []string) []Endpoint {
	var eps []Endpoint
	currentGroup := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track route groups
		if m := goGroupRe.FindStringSubmatch(trimmed); m != nil {
			currentGroup = m[1]
			continue
		}

		// Mux with .Methods()
		if m := goMuxRe.FindStringSubmatch(trimmed); m != nil {
			eps = append(eps, Endpoint{
				Method:   strings.ToUpper(m[2]),
				Path:     currentGroup + m[1],
				File:     path,
				Line:     i + 1,
				Protocol: "rest",
			})
			continue
		}

		// Router method calls: .GET, .POST, etc.
		if m := goRouterMethodRe.FindStringSubmatch(trimmed); m != nil {
			handler := ""
			if len(m) > 3 {
				handler = m[3]
			}
			eps = append(eps, Endpoint{
				Method:   strings.ToUpper(m[1]),
				Path:     currentGroup + m[2],
				Handler:  handler,
				File:     path,
				Line:     i + 1,
				Protocol: "rest",
			})
			continue
		}

		// http.HandleFunc
		if m := goHandleFuncRe.FindStringSubmatch(trimmed); m != nil {
			eps = append(eps, Endpoint{
				Method: "ANY",
				Path:   m[1],
				File:   path,
				Line:   i + 1,
				Protocol: "rest",
			})
		}
	}

	return eps
}

// --- JS/TS endpoint patterns ---

var (
	// app.get('/path', handler) — Express, Fastify, Hono
	jsRouterMethodRe = regexp.MustCompile(`\.\s*(get|post|put|delete|patch|head|options|all)\s*\(\s*['"]([^'"]+)['"]`)
	// router.route('/path')
	jsRouteRe = regexp.MustCompile(`\.route\s*\(\s*['"]([^'"]+)['"]`)
	// @Get('/path'), @Post('/path') — NestJS decorators
	jsDecoratorRe = regexp.MustCompile(`@(Get|Post|Put|Delete|Patch|Head|Options|All)\s*\(\s*['"]?([^'")]*?)['"]?\s*\)`)
	// export async function GET/POST — Next.js App Router
	jsNextRouteRe = regexp.MustCompile(`export\s+(?:async\s+)?function\s+(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\s*\(`)
)

func extractJSTSEndpoints(path string, lines []string) []Endpoint {
	var eps []Endpoint

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Next.js App Router handlers
		if m := jsNextRouteRe.FindStringSubmatch(trimmed); m != nil {
			routePath := nextRouteFromPath(path)
			eps = append(eps, Endpoint{
				Method:   strings.ToUpper(m[1]),
				Path:     routePath,
				Handler:  m[1],
				File:     path,
				Line:     i + 1,
				Protocol: "rest",
			})
			continue
		}

		// NestJS decorators
		if m := jsDecoratorRe.FindStringSubmatch(trimmed); m != nil {
			eps = append(eps, Endpoint{
				Method:   strings.ToUpper(m[1]),
				Path:     m[2],
				File:     path,
				Line:     i + 1,
				Protocol: "rest",
			})
			continue
		}

		// Express-style router methods
		if m := jsRouterMethodRe.FindStringSubmatch(trimmed); m != nil {
			eps = append(eps, Endpoint{
				Method:   strings.ToUpper(m[1]),
				Path:     m[2],
				File:     path,
				Line:     i + 1,
				Protocol: "rest",
			})
			continue
		}
	}

	return eps
}

// nextRouteFromPath converts a Next.js file path to an API route.
// e.g. "app/api/users/[id]/route.ts" → "/api/users/[id]"
func nextRouteFromPath(path string) string {
	// Strip to app/ directory
	idx := strings.Index(path, "app/")
	if idx < 0 {
		return path
	}
	route := path[idx+4:]
	// Remove route.ts/route.js/page.tsx etc.
	route = strings.TrimSuffix(route, "/route.ts")
	route = strings.TrimSuffix(route, "/route.js")
	route = strings.TrimSuffix(route, "/route.tsx")
	route = strings.TrimSuffix(route, "/route.jsx")
	if route == "" {
		return "/"
	}
	return "/" + route
}

// --- Python endpoint patterns ---

var (
	// @app.route('/path', methods=['GET'])
	pyFlaskRouteRe = regexp.MustCompile(`@\w+\.route\s*\(\s*['"]([^'"]+)['"]`)
	// @app.get('/path'), @app.post('/path') — Flask 2.0+ / FastAPI
	pyMethodRouteRe = regexp.MustCompile(`@\w+\.(get|post|put|delete|patch|head|options)\s*\(\s*['"]([^'"]+)['"]`)
	// path('url/', view) — Django
	pyDjangoPathRe = regexp.MustCompile(`(?:path|re_path|url)\s*\(\s*['"]([^'"]+)['"]`)
)

func extractPythonEndpoints(path string, lines []string) []Endpoint {
	var eps []Endpoint

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// FastAPI/Flask method decorators
		if m := pyMethodRouteRe.FindStringSubmatch(trimmed); m != nil {
			eps = append(eps, Endpoint{
				Method:   strings.ToUpper(m[1]),
				Path:     m[2],
				File:     path,
				Line:     i + 1,
				Protocol: "rest",
			})
			continue
		}

		// Flask @app.route
		if m := pyFlaskRouteRe.FindStringSubmatch(trimmed); m != nil {
			eps = append(eps, Endpoint{
				Method: "ANY",
				Path:   m[1],
				File:   path,
				Line:   i + 1,
				Protocol: "rest",
			})
			continue
		}

		// Django path()
		if m := pyDjangoPathRe.FindStringSubmatch(trimmed); m != nil {
			eps = append(eps, Endpoint{
				Method: "ANY",
				Path:   "/" + m[1],
				File:   path,
				Line:   i + 1,
				Protocol: "rest",
			})
			continue
		}
	}

	return eps
}

// --- Java endpoint patterns ---

var (
	// @GetMapping("/path"), @PostMapping, etc.
	javaSpringMappingRe = regexp.MustCompile(`@(Get|Post|Put|Delete|Patch|Request)Mapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`)
	// @RequestMapping(value="/path", method=RequestMethod.GET)
	javaRequestMappingRe = regexp.MustCompile(`@RequestMapping\s*\(.*?["']([^"']+)["'].*?method\s*=\s*RequestMethod\.(\w+)`)
	// @Path("/path") — JAX-RS
	javaJAXRSPathRe = regexp.MustCompile(`@Path\s*\(\s*["']([^"']+)["']\s*\)`)
	// @GET, @POST — JAX-RS
	javaJAXRSMethodRe = regexp.MustCompile(`@(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\b`)
)

func extractJavaEndpoints(path string, lines []string) []Endpoint {
	var eps []Endpoint
	currentClassPath := ""
	pendingMethod := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Spring @*Mapping
		if m := javaSpringMappingRe.FindStringSubmatch(trimmed); m != nil {
			method := strings.ToUpper(m[1])
			if method == "REQUEST" {
				method = "ANY"
			}
			eps = append(eps, Endpoint{
				Method:   method,
				Path:     currentClassPath + m[2],
				File:     path,
				Line:     i + 1,
				Protocol: "rest",
			})
			continue
		}

		// Spring @RequestMapping with method
		if m := javaRequestMappingRe.FindStringSubmatch(trimmed); m != nil {
			eps = append(eps, Endpoint{
				Method:   strings.ToUpper(m[2]),
				Path:     currentClassPath + m[1],
				File:     path,
				Line:     i + 1,
				Protocol: "rest",
			})
			continue
		}

		// Class-level @RequestMapping for prefix
		if strings.Contains(trimmed, "@RequestMapping") && !strings.Contains(trimmed, "method") {
			if m := regexp.MustCompile(`@RequestMapping\s*\(\s*(?:value\s*=\s*)?["']([^"']+)["']`).FindStringSubmatch(trimmed); m != nil {
				currentClassPath = m[1]
			}
			continue
		}

		// JAX-RS @Path at class level
		if m := javaJAXRSPathRe.FindStringSubmatch(trimmed); m != nil {
			// Could be class or method level
			if pendingMethod != "" {
				eps = append(eps, Endpoint{
					Method:   pendingMethod,
					Path:     currentClassPath + m[1],
					File:     path,
					Line:     i + 1,
					Protocol: "rest",
				})
				pendingMethod = ""
			} else {
				currentClassPath = m[1]
			}
			continue
		}

		// JAX-RS @GET/@POST
		if m := javaJAXRSMethodRe.FindStringSubmatch(trimmed); m != nil {
			pendingMethod = m[1]
		}
	}

	return eps
}

// --- Groovy endpoint patterns ---

var (
	// Grails URL mappings: "/api/users"(controller: "user", action: "list")
	grailsURLMappingRe = regexp.MustCompile(`["'](/[^"']+)["']\s*\(`)
	// Grails dynamic: get "/path" or post "/path"
	grailsDynRe = regexp.MustCompile(`(?:get|post|put|delete|patch)\s+["'](/[^"']+)["']`)
)

func extractGroovyEndpoints(path string, lines []string) []Endpoint {
	var eps []Endpoint

	// Only look at URL mapping files or controllers
	isURLMapping := strings.Contains(path, "UrlMappings") || strings.Contains(path, "urlmapping")
	isController := strings.HasSuffix(path, "Controller.groovy")

	if !isURLMapping && !isController {
		return nil
	}

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if m := grailsDynRe.FindStringSubmatch(trimmed); m != nil {
			method := "ANY"
			lower := strings.ToLower(trimmed)
			for _, m2 := range []string{"get", "post", "put", "delete", "patch"} {
				if strings.HasPrefix(lower, m2+" ") || strings.HasPrefix(lower, m2+"\t") {
					method = strings.ToUpper(m2)
					break
				}
			}
			eps = append(eps, Endpoint{
				Method:   method,
				Path:     m[1],
				File:     path,
				Line:     i + 1,
				Protocol: "rest",
			})
			continue
		}

		if isURLMapping {
			if m := grailsURLMappingRe.FindStringSubmatch(trimmed); m != nil {
				eps = append(eps, Endpoint{
					Method: "ANY",
					Path:   m[1],
					File:   path,
					Line:   i + 1,
					Protocol: "rest",
				})
			}
		}
	}

	return eps
}

// --- Protocol Buffers (.proto) ---

var (
	protoServiceRe = regexp.MustCompile(`service\s+(\w+)`)
	protoRPCRe     = regexp.MustCompile(`rpc\s+(\w+)\s*\(`)
)

func extractProtoEndpoints(path string, lines []string) []Endpoint {
	var eps []Endpoint
	currentService := ""

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if m := protoServiceRe.FindStringSubmatch(trimmed); m != nil {
			currentService = m[1]
			continue
		}

		if m := protoRPCRe.FindStringSubmatch(trimmed); m != nil {
			rpcPath := currentService + "/" + m[1]
			eps = append(eps, Endpoint{
				Method:   "RPC",
				Path:     rpcPath,
				Handler:  m[1],
				File:     path,
				Line:     i + 1,
				Protocol: "grpc",
			})
		}
	}

	return eps
}

// --- GraphQL (.graphql/.gql) ---

var (
	gqlTypeRe  = regexp.MustCompile(`type\s+(Query|Mutation|Subscription)\s*\{`)
	gqlFieldRe = regexp.MustCompile(`^\s*(\w+)\s*(?:\([^)]*\))?\s*:`)
)

func extractGraphQLEndpoints(path string, lines []string) []Endpoint {
	var eps []Endpoint
	currentType := ""
	inBlock := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if m := gqlTypeRe.FindStringSubmatch(trimmed); m != nil {
			currentType = m[1]
			inBlock = 1
			continue
		}

		if currentType != "" {
			inBlock += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			if inBlock <= 0 {
				currentType = ""
				continue
			}

			if m := gqlFieldRe.FindStringSubmatch(line); m != nil {
				method := "QUERY"
				if currentType == "Mutation" {
					method = "MUTATION"
				} else if currentType == "Subscription" {
					method = "SUBSCRIPTION"
				}
				eps = append(eps, Endpoint{
					Method:   method,
					Path:     m[1],
					Handler:  m[1],
					File:     path,
					Line:     i + 1,
					Protocol: "graphql",
				})
			}
		}
	}

	return eps
}
