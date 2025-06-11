// Defines the source code analyzer that validates bound checks.

package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"

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
// that validates capacity or length, or within a range statement using the range index.
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
		ident, ok := getIdentForSliceOrArr(n, pass)
		if !ok {
			return true
		}

		capCheck := false
		for i := 0; i < len(stack); i++ {
			//#nosec G602
			switch x := stack[i].(type) {
			case *ast.IfStmt: // Found an if statement
				if isIfCapCheck(x, ident) {
					capCheck = true // Was a capacity check, so this expression is fine
					break
				}
			case *ast.RangeStmt: // Found a range statement
				if isRangeIndexAccess(x, ident, n) {
					capCheck = true // Safe range index access
					break
				}
			case *ast.FuncDecl: // Found a function declaration
				if isSortInterfaceMethodAccess(x, n, pass) {
					capCheck = true // Safe sort interface method access
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
func getIdentForSliceOrArr(node ast.Node, pass *analysis.Pass) (*ast.Ident, bool) {
	switch n := node.(type) {
	case *ast.IndexExpr:
		if ident, ok := n.X.(*ast.Ident); ok {
			if typeInfo := pass.TypesInfo.TypeOf(n.X); typeInfo != nil {
				switch typeInfo.Underlying().(type) {
				case *types.Map:
					return nil, false // Skip map indexing
				case *types.Slice, *types.Array:
					return ident, true // Allow slice and array indexing
				default:
					return nil, false // Skip other types
				}
			}
			return nil, false
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
// in the call expression, false otherwise.
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

// isRangeIndexAccess checks if the given index expression is safely accessed within a range statement.
func isRangeIndexAccess(rangeStmt *ast.RangeStmt, ident *ast.Ident, indexNode ast.Node) bool {
	if rangeX, ok := rangeStmt.X.(*ast.Ident); ok {
		if rangeX.Name != ident.Name {
			return false
		}
	} else {
		return false
	}
	if indexExpr, ok := indexNode.(*ast.IndexExpr); ok {
		if indexIdent, ok := indexExpr.Index.(*ast.Ident); ok {
			if rangeKey, ok := rangeStmt.Key.(*ast.Ident); ok {
				return indexIdent.Name == rangeKey.Name
			}
		}
	}
	return false
}

// isSortInterfaceMethodAccess checks if the given index expression is accessed within a Less or Swap method
// of a type that implements the sort interface, and if the index comes from method parameters.
func isSortInterfaceMethodAccess(funcDecl *ast.FuncDecl, indexNode ast.Node, pass *analysis.Pass) bool {
	// Check if this is a Less or Swap method
	if funcDecl.Name.Name != "Less" && funcDecl.Name.Name != "Swap" {
		return false
	}

	// Must be a method (have a receiver)
	if funcDecl.Recv == nil || len(funcDecl.Recv.List) == 0 {
		return false
	}

	// Check if the receiver type implements the sort interface with correct signatures
	if !implementsSortInterface(funcDecl.Recv.List[0].Type, pass) {
		return false
	}

	// Check if the index expression uses a parameter from the method
	if indexExpr, ok := indexNode.(*ast.IndexExpr); ok {
		if indexIdent, ok := indexExpr.Index.(*ast.Ident); ok {
			return isMethodParameter(indexIdent, funcDecl)
		}
	}

	return false
}

// implementsSortInterface checks if the receiver type implements the sort interface with correct signatures.
func implementsSortInterface(recvType ast.Expr, pass *analysis.Pass) bool {
	typeObj := pass.TypesInfo.TypeOf(recvType)
	if typeObj == nil {
		return false
	}
	if ptr, ok := typeObj.(*types.Pointer); ok {
		typeObj = ptr.Elem()
	}
	named, ok := typeObj.(*types.Named)
	if !ok {
		return false
	}
	hasLen := hasMethodWithSignature(named, "Len", []string{}, []string{"int"})
	hasLess := hasMethodWithSignature(named, "Less", []string{"int", "int"}, []string{"bool"})
	hasSwap := hasMethodWithSignature(named, "Swap", []string{"int", "int"}, []string{})
	return hasLen && hasLess && hasSwap
}

// hasMethodWithSignature checks if a named type has a method with the exact signature.
func hasMethodWithSignature(named *types.Named, methodName string, paramTypes []string, returnTypes []string) bool {
	for i := 0; i < named.NumMethods(); i++ {
		method := named.Method(i)
		if method.Name() == methodName {
			sig, ok := method.Type().(*types.Signature)
			if !ok {
				continue
			}
			if signatureMatches(sig, paramTypes, returnTypes) {
				return true
			}
		}
	}
	return false
}

// signatureMatches checks if a signature matches the expected parameter and return types.
func signatureMatches(sig *types.Signature, paramTypes []string, returnTypes []string) bool {
	if sig.Params().Len() != len(paramTypes) {
		return false
	}

	for j := 0; j < sig.Params().Len(); j++ {
		paramType := sig.Params().At(j).Type()
		//#nosec G602
		if paramType.String() != paramTypes[j] {
			return false
		}
	}

	if sig.Results().Len() != len(returnTypes) {
		return false
	}

	for k := 0; k < sig.Results().Len(); k++ {
		resultType := sig.Results().At(k).Type()
		//#nosec G602
		if resultType.String() != returnTypes[k] {
			return false
		}
	}

	return true
}

// isMethodParameter checks if the given identifier is a parameter of the method.
func isMethodParameter(ident *ast.Ident, funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.Params == nil {
		return false
	}

	for _, param := range funcDecl.Type.Params.List {
		for _, name := range param.Names {
			if name.Name == ident.Name {
				return true
			}
		}
	}
	return false
}
