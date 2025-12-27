package merger

import (
	"os"
	"regexp"
	"strings"
)

const marker = "// --- RESTGEN MARKER (do not edit above) ---"
const removedMarker = "// --- REMOVED HANDLERS ---"

// Merger handles merging generated code with existing implementations.
type Merger struct{}

// New creates a new merger.
func New() *Merger {
	return &Merger{}
}

// MergeResult contains the merged output and metadata.
type MergeResult struct {
	Content          string
	PreservedMethods []string
	RemovedMethods   []string
}

// Merge combines newly generated routes with existing implementations.
// It preserves user-written handler implementations and moves removed handlers
// to the REMOVED section.
func (m *Merger) Merge(generated, existingPath string) (*MergeResult, error) {
	existing, err := os.ReadFile(existingPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No existing file, use generated as-is
			return &MergeResult{Content: generated}, nil
		}
		return nil, err
	}

	return m.MergeContent(generated, string(existing))
}

// MergeContent merges generated content with existing content.
func (m *Merger) MergeContent(generated, existing string) (*MergeResult, error) {
	result := &MergeResult{}

	// Split existing file at marker
	_, existingBelow := splitAtMarker(existing)
	generatedAbove, generatedBelow := splitAtMarker(generated)

	// Extract method implementations from existing below-marker content
	existingMethods := extractMethods(existingBelow)
	generatedMethods := extractMethods(generatedBelow)

	// Extract existing removed section
	existingRemoved := extractRemovedSection(existingBelow)

	// Build new method names set
	newMethodNames := make(map[string]bool)
	for name := range generatedMethods {
		newMethodNames[name] = true
	}

	// Preserve implementations for methods that still exist in schema
	preservedMethods := make(map[string]string)
	for name, impl := range existingMethods {
		if newMethodNames[name] {
			// Method still in schema - check if user has implemented it
			if !isStubImplementation(impl) {
				preservedMethods[name] = impl
				result.PreservedMethods = append(result.PreservedMethods, name)
			}
		} else {
			// Method removed from schema - move to removed section
			existingRemoved = append(existingRemoved, methodBlock{name: name, content: impl})
			result.RemovedMethods = append(result.RemovedMethods, name)
		}
	}

	// Build final below-marker content
	var belowMarker strings.Builder
	belowMarker.WriteString("\n\n// ============================================================================\n")
	belowMarker.WriteString("// HANDLER IMPLEMENTATIONS\n")
	belowMarker.WriteString("// ============================================================================\n")

	// Write methods in order from generated, using preserved implementations where available
	for _, m := range extractMethodsOrdered(generatedBelow) {
		if preserved, ok := preservedMethods[m.name]; ok {
			belowMarker.WriteString("\n")
			belowMarker.WriteString(preserved)
		} else {
			belowMarker.WriteString("\n")
			belowMarker.WriteString(m.content)
		}
	}

	// Write removed section
	belowMarker.WriteString("\n\n")
	belowMarker.WriteString(removedMarker)
	for _, rm := range existingRemoved {
		belowMarker.WriteString("\n\n// " + rm.name + " was removed from schema")
		belowMarker.WriteString("\n// Preserved implementation:")
		belowMarker.WriteString("\n/*\n")
		belowMarker.WriteString(rm.content)
		belowMarker.WriteString("\n*/")
	}

	// Preserve user-edited applyMiddleware and RouteMiddleware
	applyMw := extractFunction(existing, "applyMiddleware")
	routeMw := extractFunction(existing, "RouteMiddleware")

	// Rebuild above-marker with preserved middleware functions
	aboveContent := generatedAbove
	if applyMw != "" && !isDefaultApplyMiddleware(applyMw) {
		aboveContent = replaceFunction(aboveContent, "applyMiddleware", applyMw)
	}
	if routeMw != "" && !isDefaultRouteMiddleware(routeMw) {
		aboveContent = replaceFunction(aboveContent, "RouteMiddleware", routeMw)
	}

	// Also preserve handler struct fields
	existingStruct := extractHandlerStruct(existing)
	if existingStruct != "" && !isEmptyHandlerStruct(existingStruct) {
		aboveContent = replaceHandlerStruct(aboveContent, existingStruct)
	}

	result.Content = aboveContent + "\n" + marker + belowMarker.String()
	return result, nil
}

func splitAtMarker(content string) (above, below string) {
	idx := strings.Index(content, marker)
	if idx == -1 {
		return content, ""
	}
	return content[:idx], content[idx+len(marker):]
}

type methodBlock struct {
	name    string
	content string
}

// extractMethods extracts func (h *Handler) MethodName(...) implementations.
func extractMethods(content string) map[string]string {
	methods := make(map[string]string)

	for _, m := range extractMethodsOrdered(content) {
		methods[m.name] = m.content
	}

	return methods
}

// extractMethodsOrdered returns methods in order of appearance using brace matching.
func extractMethodsOrdered(content string) []methodBlock {
	var methods []methodBlock

	// Find "func (h *SomethingHandler) MethodName("
	re := regexp.MustCompile(`func \(h \*\w+Handler\) (\w+)\(`)
	matches := re.FindAllStringSubmatchIndex(content, -1)

	for _, match := range matches {
		methodName := content[match[2]:match[3]]
		funcStart := match[0]

		// Find the opening brace of the function body
		braceIdx := strings.Index(content[funcStart:], "{")
		if braceIdx == -1 {
			continue
		}
		braceIdx += funcStart

		// Find matching closing brace
		depth := 1
		bodyEnd := braceIdx + 1

		for i := braceIdx + 1; i < len(content) && depth > 0; i++ {
			if content[i] == '{' {
				depth++
			} else if content[i] == '}' {
				depth--
				if depth == 0 {
					bodyEnd = i + 1
				}
			}
		}

		methods = append(methods, methodBlock{
			name:    methodName,
			content: content[funcStart:bodyEnd],
		})
	}

	return methods
}

func extractRemovedSection(content string) []methodBlock {
	idx := strings.Index(content, removedMarker)
	if idx == -1 {
		return nil
	}

	// Parse commented-out methods in removed section
	removed := content[idx+len(removedMarker):]
	var methods []methodBlock

	// Match: // MethodName was removed from schema ... /* ... */
	re := regexp.MustCompile(`// (\w+) was removed from schema[^/]*/\*\s*(func [^*]+)\*/`)
	matches := re.FindAllStringSubmatch(removed, -1)

	for _, m := range matches {
		methods = append(methods, methodBlock{
			name:    m[1],
			content: strings.TrimSpace(m[2]),
		})
	}

	return methods
}

func isStubImplementation(impl string) bool {
	return strings.Contains(impl, "TODO: implement") &&
		strings.Contains(impl, "StatusNotImplemented")
}

func extractFunction(content, funcName string) string {
	// Find "func (h *SomethingHandler) funcName("
	re := regexp.MustCompile(`func \(h \*\w+Handler\) ` + funcName + `\(`)
	loc := re.FindStringIndex(content)
	if loc == nil {
		return ""
	}

	funcStart := loc[0]

	// Find the opening brace
	braceIdx := strings.Index(content[funcStart:], "{")
	if braceIdx == -1 {
		return ""
	}
	braceIdx += funcStart

	// Find matching closing brace
	depth := 1
	bodyEnd := braceIdx + 1

	for i := braceIdx + 1; i < len(content) && depth > 0; i++ {
		if content[i] == '{' {
			depth++
		} else if content[i] == '}' {
			depth--
			if depth == 0 {
				bodyEnd = i + 1
			}
		}
	}

	return content[funcStart:bodyEnd]
}

func isDefaultApplyMiddleware(impl string) bool {
	// Check if it's still the default generated stub
	return strings.Contains(impl, "// Example:") &&
		!strings.Contains(impl, "r.Use(")
}

func isDefaultRouteMiddleware(impl string) bool {
	return strings.Contains(impl, `// "POST /":`)
}

func replaceFunction(content, funcName, newImpl string) string {
	re := regexp.MustCompile(`(?s)func \(h \*\w+Handler\) ` + funcName + `\([^)]*\)[^{]*\{[^}]*(?:\{[^}]*\}[^}]*)*\}`)
	return re.ReplaceAllString(content, newImpl)
}

func extractHandlerStruct(content string) string {
	re := regexp.MustCompile(`(?s)(type \w+Handler struct \{[^}]*\})`)
	m := re.FindStringSubmatch(content)
	if len(m) > 1 {
		return m[1]
	}
	return ""
}

func isEmptyHandlerStruct(structDef string) bool {
	return strings.Contains(structDef, "// add dependencies here")
}

func replaceHandlerStruct(content, newStruct string) string {
	re := regexp.MustCompile(`(?s)type \w+Handler struct \{[^}]*\}`)
	return re.ReplaceAllString(content, newStruct)
}
