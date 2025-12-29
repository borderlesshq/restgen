package merger

import (
	"go/ast"
	"go/parser"
	"go/token"
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
// MergeContent merges generated content with existing content.
func (m *Merger) MergeContent(generated, existing string) (*MergeResult, error) {
	result := &MergeResult{}

	// Split existing file at marker
	existingAbove, existingBelow := splitAtMarker(existing)
	generatedAbove, generatedBelow := splitAtMarker(generated)

	// Preserve handler struct fields from existing above-marker content
	mergedAbove := preserveHandlerStructFields(generatedAbove, existingAbove)

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

	// Preserve ALL existing methods that still exist in schema
	// Use AST to check if the method body is non-trivial (not just a stub)
	preservedMethods := make(map[string]string)
	for name, impl := range existingMethods {
		if newMethodNames[name] {
			// Method still in schema - always preserve if it has real code
			if !isGeneratedStub(impl) {
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

	result.Content = mergedAbove + "\n" + marker + belowMarker.String()
	return result, nil
}

// preserveHandlerStructFields extracts the handler struct from existing content
// and merges its fields into the generated content.
func preserveHandlerStructFields(generated, existing string) string {
	// Extract handler struct from existing
	existingStruct := extractHandlerStruct(existing)
	if existingStruct == "" {
		return generated
	}

	// Check if existing struct has custom fields (not just the comment)
	if isEmptyHandlerStruct(existingStruct) {
		return generated
	}

	// Extract handler struct from generated
	generatedStruct := extractHandlerStruct(generated)
	if generatedStruct == "" {
		return generated
	}

	// Replace the generated struct with the existing one
	return strings.Replace(generated, generatedStruct, existingStruct, 1)
}

// extractHandlerStruct extracts the handler struct definition including its body.
// Matches: type XxxHandler struct { ... }
func extractHandlerStruct(content string) string {
	re := regexp.MustCompile(`type\s+\w+Handler\s+struct\s*\{`)
	match := re.FindStringIndex(content)
	if match == nil {
		return ""
	}

	start := match[0]
	braceStart := match[1] - 1

	// Find matching closing brace
	depth := 1
	end := braceStart + 1
	for i := braceStart + 1; i < len(content) && depth > 0; i++ {
		if content[i] == '{' {
			depth++
		} else if content[i] == '}' {
			depth--
			if depth == 0 {
				end = i + 1
			}
		}
	}

	return content[start:end]
}

// isEmptyHandlerStruct checks if the struct only contains the default comment.
func isEmptyHandlerStruct(structDef string) bool {
	// Remove the struct wrapper
	inner := structDef
	if idx := strings.Index(inner, "{"); idx != -1 {
		inner = inner[idx+1:]
	}
	if idx := strings.LastIndex(inner, "}"); idx != -1 {
		inner = inner[:idx]
	}

	// Trim whitespace and check if only contains the default comment or is empty
	inner = strings.TrimSpace(inner)
	return inner == "" ||
		inner == "// add dependencies here" ||
		strings.HasPrefix(inner, "// add dependencies here") && strings.TrimSpace(strings.TrimPrefix(inner, "// add dependencies here")) == ""
}

// isGeneratedStub uses Go AST to check if a method is an unmodified generated stub.
// A stub has exactly the pattern:
//   - Optional: commented decode code (comments are ignored by AST)
//   - Optional: var declaration + if decode error block
//   - A single WriteResponse call with StatusNotImplemented
func isGeneratedStub(impl string) bool {
	// Quick check: if it doesn't have the stub markers, it's not a stub
	if !strings.Contains(impl, "StatusNotImplemented") {
		return false
	}

	// Wrap the method in a package to make it parseable
	src := "package stub\n" + impl

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, parser.ParseComments)
	if err != nil {
		// If we can't parse it, assume it's been modified and preserve it
		return false
	}

	// Find the function declaration
	var funcDecl *ast.FuncDecl
	for _, decl := range f.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			funcDecl = fd
			break
		}
	}

	if funcDecl == nil || funcDecl.Body == nil {
		return false
	}

	// Analyze the statements to determine if this is a stub
	// A generated stub has a specific pattern:
	// 1. Optional: var decl + if decode error (for body/query params)
	// 2. Final: WriteResponse with StatusNotImplemented

	statements := funcDecl.Body.List
	if len(statements) == 0 {
		return false
	}

	// Check if the last statement is the NotImplemented response
	lastStmt := statements[len(statements)-1]
	if !isNotImplementedResponse(lastStmt) {
		return false
	}

	// Check all other statements - they should only be decode-related
	for i := 0; i < len(statements)-1; i++ {
		if !isDecodeRelatedStatement(statements[i]) {
			// Found a statement that's not part of the template
			return false
		}
	}

	return true
}

// isNotImplementedResponse checks if a statement is the WriteResponse with StatusNotImplemented
func isNotImplementedResponse(stmt ast.Stmt) bool {
	exprStmt, ok := stmt.(*ast.ExprStmt)
	if !ok {
		return false
	}

	call, ok := exprStmt.X.(*ast.CallExpr)
	if !ok {
		return false
	}

	// Check if it's a WriteResponse call
	switch fn := call.Fun.(type) {
	case *ast.SelectorExpr:
		if fn.Sel.Name != "WriteResponse" {
			return false
		}
	case *ast.Ident:
		if fn.Name != "writeResponse" && fn.Name != "WriteResponse" {
			return false
		}
	default:
		return false
	}

	// Check if second argument is http.StatusNotImplemented
	if len(call.Args) >= 2 {
		if sel, ok := call.Args[1].(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "StatusNotImplemented" {
				return true
			}
		}
	}

	return false
}

// isDecodeRelatedStatement checks if a statement is part of the generated decode template
func isDecodeRelatedStatement(stmt ast.Stmt) bool {
	switch s := stmt.(type) {
	case *ast.DeclStmt:
		// var declarations for decode targets (var input models.X)
		return true
	case *ast.AssignStmt:
		// decoder := schema.NewDecoder() or similar
		for _, rhs := range s.Rhs {
			if containsDecoderSetup(rhs) {
				return true
			}
		}
		return false
	case *ast.ExprStmt:
		// decoder.IgnoreUnknownKeys(true) or similar
		if call, ok := s.X.(*ast.CallExpr); ok {
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
				if sel.Sel.Name == "IgnoreUnknownKeys" {
					return true
				}
			}
		}
		return false
	case *ast.IfStmt:
		// if err := decoder.Decode(...); err != nil { WriteResponse(...) return }
		return isDecodeErrorIf(s)
	}
	return false
}

// isDecodeErrorIf checks if an if statement is the decode error handling pattern
func isDecodeErrorIf(stmt *ast.IfStmt) bool {
	// Pattern: if err := json.NewDecoder(...).Decode(...); err != nil { ... return }
	// or: if err := decoder.Decode(...); err != nil { ... return }
	if stmt.Init == nil {
		return false
	}

	assign, ok := stmt.Init.(*ast.AssignStmt)
	if !ok {
		return false
	}

	// Check if RHS contains Decode call
	for _, rhs := range assign.Rhs {
		if containsDecodeCall(rhs) {
			// Also verify the body ends with return
			if stmt.Body != nil && len(stmt.Body.List) > 0 {
				lastStmt := stmt.Body.List[len(stmt.Body.List)-1]
				if _, ok := lastStmt.(*ast.ReturnStmt); ok {
					return true
				}
			}
		}
	}
	return false
}

// containsDecoderSetup checks if an expression is decoder setup
func containsDecoderSetup(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.CallExpr:
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "NewDecoder" {
				return true
			}
		}
	}
	return false
}

// containsDecodeCall recursively checks if an expression contains a Decode call
func containsDecodeCall(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.CallExpr:
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "Decode" {
				return true
			}
		}
		// Check nested calls (e.g., json.NewDecoder(r.Body).Decode(&x))
		if containsDecodeCall(e.Fun) {
			return true
		}
		for _, arg := range e.Args {
			if containsDecodeCall(arg) {
				return true
			}
		}
	case *ast.SelectorExpr:
		return containsDecodeCall(e.X)
	}
	return false
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
