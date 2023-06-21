package analyzer

import (
	"go/ast"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "goboundcheck",
	Doc:      "Checks that slice and array access is not out of bounds.",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	inspector := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// We are looking for index and slice expressions to validate.
	nodeFilter := []ast.Node{
		(*ast.IndexExpr)(nil),
		(*ast.SliceExpr)(nil),
	}

	// Inspect source code depth-first with stack of visited nodes so we can get parent nodes
	inspector.WithStack(nodeFilter, func(n ast.Node, push bool, stack []ast.Node) bool {
		// Go through all parents and see if it checks capacity.
		capCheck := false
		for i := 0; i < len(stack); i++ {
			switch x := stack[i].(type) {
			case *ast.IfStmt: // Found an if statement
				if isIfCapCheck(x) {
					capCheck = true // Was a capacity check, so this expression is fine
					break
				}
			}
		}

		if !capCheck {
			pass.Reportf(n.Pos(), "Slice or array access is not enclosed in an if-statement that validates capacity!")
		}

		return true
	})

	return nil, nil
}

func isIfCapCheck(ifStmt *ast.IfStmt) bool {
	cond := ifStmt.Cond // If condition

	switch expr := cond.(type) {
	case *ast.BinaryExpr:
		if call, ok := expr.X.(*ast.CallExpr); ok {
			if isCapOrLenCall(call) {
				return true
			}
		}

		if call, ok := expr.Y.(*ast.CallExpr); ok {
			if isCapOrLenCall(call) {
				return true
			}
		}

		return false
	default:
		return false
	}
}

func isCapOrLenCall(call *ast.CallExpr) bool {
	if ident, ok := call.Fun.(*ast.Ident); ok {
		if ident.Name == "cap" || ident.Name == "len" {
			return true
		}
	}

	return false
}
