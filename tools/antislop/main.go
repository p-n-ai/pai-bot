package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type finding struct {
	position token.Position
	rule     string
	message  string
}

func main() {
	flag.Parse()
	roots := flag.Args()
	if len(roots) == 0 {
		roots = []string{"cmd", "internal"}
	}

	findings, err := inspect(roots)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	for _, finding := range findings {
		fmt.Printf(
			"%s:%d:%d: %s: %s\n",
			finding.position.Filename,
			finding.position.Line,
			finding.position.Column,
			finding.rule,
			finding.message,
		)
	}
	if len(findings) > 0 {
		os.Exit(1)
	}
}

func inspect(roots []string) ([]finding, error) {
	files, err := goFiles(roots)
	if err != nil {
		return nil, err
	}

	findings := make([]finding, 0)
	for _, filename := range files {
		fileSet := token.NewFileSet()
		file, parseErr := parser.ParseFile(fileSet, filename, nil, parser.ParseComments)
		if parseErr != nil {
			return nil, parseErr
		}
		if ast.IsGenerated(file) {
			continue
		}

		ast.Inspect(file, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.FuncType:
				findings = checkFunctionType(fileSet, node, findings)
			case *ast.Ident:
				findings = checkSymbolName(fileSet, node, findings)
			case *ast.MapType:
				findings = checkUnsafeDictionary(fileSet, node, findings)
			case *ast.TypeAssertExpr:
				findings = checkChainedTypeAssertion(fileSet, node, findings)
			case *ast.TypeSpec:
				findings = checkTypeSpec(fileSet, node, findings)
			case *ast.ValueSpec:
				findings = checkKnownValueWidening(fileSet, node, findings)
			}
			return true
		})
	}

	sort.Slice(findings, func(left, right int) bool {
		leftPosition := findings[left].position
		rightPosition := findings[right].position
		if leftPosition.Filename != rightPosition.Filename {
			return leftPosition.Filename < rightPosition.Filename
		}
		if leftPosition.Line != rightPosition.Line {
			return leftPosition.Line < rightPosition.Line
		}
		if leftPosition.Column != rightPosition.Column {
			return leftPosition.Column < rightPosition.Column
		}
		return findings[left].rule < findings[right].rule
	})
	return findings, nil
}

func goFiles(roots []string) ([]string, error) {
	files := make([]string, 0)
	for _, root := range roots {
		info, err := os.Stat(root)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			if isProductionGoFile(root) {
				files = append(files, filepath.Clean(root))
			}
			continue
		}

		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if entry.Name() == "vendor" || strings.HasPrefix(entry.Name(), ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if isProductionGoFile(path) {
				files = append(files, filepath.Clean(path))
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func isProductionGoFile(path string) bool {
	return filepath.Ext(path) == ".go" && !strings.HasSuffix(path, "_test.go")
}

func checkFunctionType(fileSet *token.FileSet, function *ast.FuncType, findings []finding) []finding {
	for _, field := range function.Params.List {
		parameterType := unwrapParameterType(field.Type)
		if isAnyType(parameterType) && !allNamesEqual(field.Names, "cause") {
			findings = report(
				fileSet,
				field.Type.Pos(),
				"no-unknown-parameters",
				fmt.Sprintf("parameter %s accepts any; parse boundary input into a concrete owner type", fieldNames(field.Names)),
				findings,
			)
		}
		if _, ok := parameterType.(*ast.StructType); ok {
			findings = report(
				fileSet,
				field.Type.Pos(),
				"no-object-parameters",
				fmt.Sprintf("parameter %s uses an anonymous struct; use a named owner contract", fieldNames(field.Names)),
				findings,
			)
		}
	}
	return findings
}

func checkTypeSpec(fileSet *token.FileSet, typeSpec *ast.TypeSpec, findings []finding) []finding {
	if isAnyType(typeSpec.Type) {
		findings = report(
			fileSet,
			typeSpec.Name.Pos(),
			"no-unknown-type-aliases",
			fmt.Sprintf("type %s resolves to any; keep the boundary type explicit", typeSpec.Name.Name),
			findings,
		)
	}
	return findings
}

func checkKnownValueWidening(fileSet *token.FileSet, valueSpec *ast.ValueSpec, findings []finding) []finding {
	if !isAnyType(valueSpec.Type) {
		return findings
	}
	for _, value := range valueSpec.Values {
		if isConcreteExpression(value) {
			findings = report(
				fileSet,
				value.Pos(),
				"no-known-value-widening",
				"concrete initializer is widened to any; preserve its inferred type",
				findings,
			)
		}
	}
	return findings
}

func checkChainedTypeAssertion(fileSet *token.FileSet, assertion *ast.TypeAssertExpr, findings []finding) []finding {
	if _, ok := unwrapParentheses(assertion.X).(*ast.TypeAssertExpr); !ok {
		return findings
	}
	return report(
		fileSet,
		assertion.Pos(),
		"no-chained-type-assertions",
		"chained type assertions discard evidence; parse or assert once at the boundary",
		findings,
	)
}

func checkSymbolName(fileSet *token.FileSet, identifier *ast.Ident, findings []finding) []finding {
	if identifier.Obj == nil || !strings.Contains(strings.ToLower(identifier.Name), "shape") {
		return findings
	}
	return report(
		fileSet,
		identifier.Pos(),
		"no-shape-in-symbol-names",
		fmt.Sprintf("symbol %s describes structure instead of domain meaning", identifier.Name),
		findings,
	)
}

func checkUnsafeDictionary(fileSet *token.FileSet, mapType *ast.MapType, findings []finding) []finding {
	if !isStringType(mapType.Key) || !isAnyType(mapType.Value) {
		return findings
	}
	return report(
		fileSet,
		mapType.Pos(),
		"no-unsafe-dictionary-type",
		"map[string]any hides its value contract; use a concrete owner type",
		findings,
	)
}

func report(
	fileSet *token.FileSet,
	position token.Pos,
	rule string,
	message string,
	findings []finding,
) []finding {
	return append(findings, finding{
		position: fileSet.PositionFor(position, false),
		rule:     rule,
		message:  message,
	})
}

func unwrapParameterType(expression ast.Expr) ast.Expr {
	current := expression
	for {
		switch node := current.(type) {
		case *ast.Ellipsis:
			current = node.Elt
		case *ast.ParenExpr:
			current = node.X
		case *ast.StarExpr:
			current = node.X
		default:
			return current
		}
	}
}

func unwrapParentheses(expression ast.Expr) ast.Expr {
	current := expression
	for {
		parenthesized, ok := current.(*ast.ParenExpr)
		if !ok {
			return current
		}
		current = parenthesized.X
	}
}

func isAnyType(expression ast.Expr) bool {
	switch expression := expression.(type) {
	case *ast.Ident:
		return expression.Name == "any" && expression.Obj == nil
	case *ast.InterfaceType:
		return expression.Methods != nil && len(expression.Methods.List) == 0
	case *ast.ParenExpr:
		return isAnyType(expression.X)
	default:
		return false
	}
}

func isStringType(expression ast.Expr) bool {
	identifier, ok := unwrapParentheses(expression).(*ast.Ident)
	return ok && identifier.Name == "string" && identifier.Obj == nil
}

func isConcreteExpression(expression ast.Expr) bool {
	switch expression := unwrapParentheses(expression).(type) {
	case *ast.BasicLit, *ast.CompositeLit, *ast.FuncLit:
		return true
	case *ast.UnaryExpr:
		return expression.Op != token.ARROW && isConcreteExpression(expression.X)
	default:
		return false
	}
}

func allNamesEqual(names []*ast.Ident, allowed string) bool {
	if len(names) == 0 {
		return false
	}
	for _, name := range names {
		if name.Name != allowed {
			return false
		}
	}
	return true
}

func fieldNames(names []*ast.Ident) string {
	if len(names) == 0 {
		return "<unnamed>"
	}
	values := make([]string, 0, len(names))
	for _, name := range names {
		values = append(values, name.Name)
	}
	return strings.Join(values, ", ")
}
