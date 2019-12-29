package gc

import (
	"cmd/compile/internal/syntax"
	"cmd/compile/internal/types"
	"cmd/internal/objabi"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func importPackage() {
	ipks := types.ImportedPkgList()

	for _, imp := range syntax.MissingImportDecl() {
		unquote, _ := strconv.Unquote(imp.Path.Value)

		found := false
		for _, ipk := range ipks {
			if ipk.Path == unquote {
				found = true
				break
			}
		}
		if found {
			continue
		}

		if packageFile != nil {
			if objabi.GOROOT != "" {
				suffix := ""
				suffixsep := ""
				if flag_installsuffix != "" {
					suffixsep = "_"
					suffix = flag_installsuffix
				} else if flag_race {
					suffixsep = "_"
					suffix = "race"
				} else if flag_msan {
					suffixsep = "_"
					suffix = "msan"
				}
				file := fmt.Sprintf("%s/pkg/%s_%s%s%s/%s.a", objabi.GOROOT, objabi.GOOS, objabi.GOARCH, suffixsep, suffix, unquote)
				if _, err := os.Stat(file); err != nil {
					yyerror("can't find import: %q", unquote)
					errorexit()
				}
				packageFile[unquote] = file
			}
		}

		val := Val{U: unquote}
		ipkg := importfile(&val)
		ipkg.Direct = true
		my := lookup(ipkg.Name)
		pack := nod(OPACK, nil, nil)
		pack.Sym = my
		pack.Name.Pkg = ipkg
		if my.Def != nil {
			redeclare(pack.Pos, my, "as imported package name")
		}
		my.Def = asTypesNode(pack)
		my.Lastlineno = pack.Pos
		my.Block = 1 // at top level
	}
}

func transformmagicRoof(xstmt *Node) {
	transformmagic(xstmt)
}

func transformmagic(xstmt *Node) (index int, stmt *Node) {
	if xstmt == nil {
		return -1, nil
	}
	if xstmt.MagicStmt != nil {
		defer func() {
			result := Curfn.Type.Results()
			size := result.Fields().Len()
			hasChange := false
			rets := xstmt.MagicStmt.Nbody.Index(0).List.Slice()
			for i := 0; i < size; i++ {
				fl := result.Field(i)
				if rets[i].Op != OCALL && rets[i] != xstmt.MagicExpr {
					switch fl.Type.Etype {
					case TINTER:
						if rets[i].Op == OCOMPLIT {
							hasChange = true
							rets[i] = xstmt.MagicReNil
						}

					case TFUNC:
						if rets[i].Op == OCOMPLIT {
							hasChange = true
							rets[i] = xstmt.MagicReNil
						}
					}
				}
			}
			if hasChange {
				xstmt.MagicStmt.Nbody.Index(0).List.Set(rets)
			}

			xstmt.MagicReNil = nil
			xstmt.MagicStmt = nil
			xstmt.MagicExprIndex = -1
			xstmt.MagicStmtIndex = -1
		}()

		var expr *Node
		if xstmt.MagicExprIndex < xstmt.List.Len() {
			expr = xstmt.List.Index(xstmt.MagicExprIndex)
		} else if xstmt.Op == OCASE {
			if xstmt.MagicExprIndex < xstmt.Left.List.Len() {
				expr = xstmt.Left.List.Index(xstmt.MagicExprIndex)
			} else if xstmt.Left.Op == OSELRECV2 && xstmt.Left.List.Len() > 0 {
				expr = xstmt.Left.List.Index(0)
			}
		} else if xstmt.Op == OAS {
			expr = xstmt.Left
		}
		if expr == nil {
			return -1, nil
		}

		t := expr.Type
		switch t.Etype {
		case TBOOL:
			// a boolean type
			// need to change if condition and body include panic and return turn type.
			xstmt.MagicStmt.Left = nodl(expr.Pos, ONOT, expr, nil)
			lstmt := xstmt.MagicStmt.Nbody.Index(0).List
			size := lstmt.Len()
			for i := 0; i < size; i++ {
				if lstmt.Index(i) == expr {
					lstmt.SetIndex(i, xstmt.MagicExpr)
				}
			}
			switch xstmt.Op {
			case OAS2MAPR:
				lsarg := xstmt.MagicExpr.List.Index(0)
				nodes := []*Node{lsarg.List.Index(0), nil}
				nodes[0].E = fmt.Sprintf("key %%v does not existed in map %s", xstmt.Right.Left.Sym.Name)
				nodes[1] = xstmt.Right.Right
				lsarg.List.Set(nodes)
				xstmt.MagicExpr.List.SetIndex(0, lsarg)

			case OAS2RECV:
				xstmt.MagicExpr.List.Index(0).List.Index(0).E = "channel has been closed"

			case OAS2:
				source := xstmt.Rlist.Index(xstmt.MagicExprIndex)
				switch source.Op {
				case OCALLFUNC:
					xstmt.MagicExpr.List.Index(0).List.Index(0).E = fmt.Sprintf("call to %s return false", source.Left.Sym.Name)
				}

			case OAS2FUNC:
				switch xstmt.Right.Op {
				case OCALLFUNC:
					xstmt.MagicExpr.List.Index(0).List.Index(0).E = fmt.Sprintf("call to %s return false", xstmt.Right.Left.Sym.Name)
				}

			case OAS2DOTTYPE:
				switch xstmt.Right.Op {
				case ODOTTYPE2:
					castType := strings.ToLower(xstmt.Right.Type.Etype.String())
					xstmt.MagicExpr.List.Index(0).List.Index(0).E = fmt.Sprintf("cannot cast %s to %s", xstmt.Right.Left.Sym.Name, castType)
				}

			case OCASE:
				// select case
				body := append([]*Node{xstmt.MagicStmt}, xstmt.Nbody.Slice()...)
				xstmt.Nbody.Set(body)
				typecheck(xstmt.MagicStmt, 1)
				return -1, nil

			}

			return xstmt.MagicStmtIndex, xstmt.MagicStmt

		case TINTER:
			if t.Methods().Len() == 1 {
				field := t.Methods().Index(0)
				if field.Sym != nil && field.Sym.Name == "Error" {
					// implement error interface
					// We do not need to change anything
					xstmt.MagicExpr = expr
					return xstmt.MagicStmtIndex, xstmt.MagicStmt
				}
			}
		}

		yyerrorl(xstmt.Pos, "magic variable must be a boolean or an error")

	}

	deep := func(body []*Node) []*Node {
		index := make([]int, 0)
		stmts := make([]*Node, 0)

		for _, n := range body {
			if i, stmt := transformmagic(n); i >= 0 {
				index = append(index, i)
				stmts = append(stmts, stmt)
			}
		}

		if i := len(index); i > 0 {
			for i -= 1; i >= 0; i-- {
				idx := index[i]
				body = append(body[:idx], append([]*Node{stmts[i]}, body[idx:]...)...)
				typecheck(stmts[i], 1)
			}
			return body
		}
		return nil
	}

	if xstmt.Nbody.Len() > 0 {
		if body := deep(xstmt.Nbody.Slice()); body != nil {
			xstmt.Nbody.Set(body)
		}
	}

	if xstmt.Rlist.Len() > 0 {
		if body := deep(xstmt.Rlist.Slice()); body != nil {
			xstmt.Rlist.Set(body)
		}
	}

	if xstmt.List.Len() > 0 {
		if body := deep(xstmt.List.Slice()); body != nil {
			xstmt.List.Set(body)
		}
	}

	if xstmt.Right != nil && xstmt.Right.Op == OCLOSURE {
		Previous := Curfn
		Curfn = xstmt.Right.Func.Closure
		if xstmt.Right.Func.Closure.Nbody.Len() > 0 {
			if body := deep(xstmt.Right.Func.Closure.Nbody.Slice()); body != nil {
				xstmt.Right.Func.Closure.Nbody.Set(body)
			}
		}
		Curfn = Previous
	}

	return -1, nil
}

func createNodeFormatMessage(s string, nodes ...*Node) *Node {
	return nod(OCALLFUNC, nod(ONAME, nil, nil), nil)
}
