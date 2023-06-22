// Defines the source code analyzer that validates bound checks.

package analyzer

import (
	"go/ast"
	"go/token"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// Analyzer describes the analysis function for bound checking.
var Analyzer = &analysis.Analyzer{
	Name:     "goboundcheck",
	Doc:      "Checks that slice and array access is not out of bounds.",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

// run scans all index and slice expressions for any accesses which are not enclosed within an if-statement
// that validates capacity or length.
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
		ident, ok := getIdentForSliceOrArr(n)
		if !ok {
			return true
		}

		capCheck := false
		for i := 0; i < len(stack); i++ {
			switch x := stack[i].(type) {
			case *ast.IfStmt: // Found an if statement
				if isIfCapCheck(x, ident) {
					capCheck = true // Was a capacity check, so this expression is fine
					break
				}
			}
		}

		if !capCheck {
			pass.Reportf(n.Pos(), "Slice or array access is not enclosed in an if-statement that validates capacity!")
			return false
		}

		return true
	})

	return nil, nil
}

// getIdentForSliceOrArr takes an index or slice expr node and returns the underlying ident that
// the expression refers to and true on success, nil and false on failure.
func getIdentForSliceOrArr(node ast.Node) (*ast.Ident, bool) {
	switch n := node.(type) {
	case *ast.IndexExpr:
		if ident, ok := n.X.(*ast.Ident); ok {
			return ident, true
		} else {
			return nil, false
		}
	case *ast.SliceExpr:
		if ident, ok := n.X.(*ast.Ident); ok {
			return ident, true
		} else {
			return nil, false
		}
	default:
		return nil, false
	}
}

func binExprHasLenCapCall(bin *ast.BinaryExpr, ident *ast.Ident) bool {
	if bin.Op == token.LOR || bin.Op == token.LAND || bin.Op == token.LEQ {
		hasCapLenCall := false
		if binExprX, ok := bin.X.(*ast.BinaryExpr); ok {
			if binExprHasLenCapCall(binExprX, ident) {
				hasCapLenCall = true
			}
		}
		if binExprY, ok := bin.Y.(*ast.BinaryExpr); ok {
			if binExprHasLenCapCall(binExprY, ident) {
				hasCapLenCall = true
			}
		}

		return hasCapLenCall
	} else {
		if call, ok := bin.X.(*ast.CallExpr); ok {
			// Check that its a cap/len call and has ident as an arg
			if isCapOrLenCall(call) && isIdentFuncArg(call, ident) {
				return true
			}
		}

		if call, ok := bin.Y.(*ast.CallExpr); ok {
			if isCapOrLenCall(call) && isIdentFuncArg(call, ident) {
				return true
			}
		}
	}
	return false
}

// ifIsCapCheck takes an if-statemnt and the ident for the variable being accessed. It checks that
// the given if-statement compares the capacity or length of the given variable as a condition. Returns
// true if it is a valid check, false if not or if there is no check at all.
func isIfCapCheck(ifStmt *ast.IfStmt, ident *ast.Ident) bool {
	cond := ifStmt.Cond // If condition

	switch expr := cond.(type) {
	case *ast.BinaryExpr:
		return binExprHasLenCapCall(expr, ident)
	default:
		return false
	}
}

// isIdentFuncArg takes a call expression and an ident. It returns true if the given ident is an argument
// in the call expression, false otherwise
func isIdentFuncArg(call *ast.CallExpr, ident *ast.Ident) bool {
	if call.Args == nil {
		return false
	}

	// Go through each arg and compare the ident with the given ident
	for _, arg := range call.Args {
		switch a := arg.(type) {
		case *ast.Ident:
			if a.Name == ident.Name {
				return true
			}
		}
	}

	return false
}

// isCapOrLenCall takes a call as input and returns true if the call is a call to cap or len, false if otherwise.
func isCapOrLenCall(call *ast.CallExpr) bool {
	if ident, ok := call.Fun.(*ast.Ident); ok {
		if ident.Name == "cap" || ident.Name == "len" {
			return true
		}
	}

	return false
}
