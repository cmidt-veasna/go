package syntax

import (
	"fmt"
	"strings"
)

func CreateIfMagicStatement(ex Expr, me *MagicExpr, res []string) *IfStmt {
	n := node{pos: ex.Pos()}
	posEx := expr{node: n}
	posSt := stmt{node: n}
	name := &Name{Value: ex.(*Name).Value, expr: posEx}
	nilName := &Name{Value: "nil", expr: posEx}

	var bodyStmt Stmt
	if me.Panic {
		// the name will be will replace with appropriate type variable if variable is not
		// an error type
		bodyStmt = &ExprStmt{
			X: &CallExpr{
				Fun:     &Name{Value: "panic", expr: posEx},
				ArgList: []Expr{name},
				HasDots: false,
				expr:    posEx,
			},
			simpleStmt: simpleStmt{stmt{n}},
		}
	} else {
		var result Expr
		if len(res) == 1 {
			// this will replace let one with appropriate type variable if variable is not
			// an error type
			result = name
		} else if len(res) > 1 {
			list := &ListExpr{expr: posEx}
			result = list
			for _, t := range res {
				switch t {
				case "error":
					// this will replace let one with appropriate type variable if variable is not
					// an error type
					list.ElemList = append(list.ElemList, name)

				case "string":
					list.ElemList = append(list.ElemList, &BasicLit{Value: `""`, Kind: StringLit, expr: posEx})

				case "bool":
					list.ElemList = append(list.ElemList, &Name{Value: "false", expr: posEx})

				default:
					if strings.HasPrefix(t, "*") || strings.HasPrefix(t, "[") ||
						strings.HasPrefix(t, "map") || strings.HasPrefix(t, "interface{}") ||
						strings.HasPrefix(t, "func") {
						list.ElemList = append(list.ElemList, nilName)

					} else if strings.Contains(t, "complex") {
						list.ElemList = append(list.ElemList, &CallExpr{
							Fun: &Name{Value: "complex", expr: posEx},
							ArgList: []Expr{
								&BasicLit{
									Value: "0",
									Kind:  FloatLit,
									expr:  posEx,
								},
								&BasicLit{
									Value: "0",
									Kind:  FloatLit,
									expr:  posEx,
								},
							},
							expr: posEx,
						})

					} else if strings.Contains(t, "int") || t == "byte" {
						list.ElemList = append(list.ElemList, &BasicLit{Value: "0", Kind: IntLit, expr: posEx})
					} else if strings.Contains(t, "float") {
						list.ElemList = append(list.ElemList, &BasicLit{Value: "0", Kind: FloatLit, expr: posEx})
					} else if t == "rune" {
						list.ElemList = append(list.ElemList, &BasicLit{Value: "0", Kind: RuneLit, expr: posEx})
					} else {
						dotIndex := strings.IndexByte(t, '.')
						var reexpr Expr
						if dotIndex > 0 {
							parts := strings.Split(t, ".")
							reexpr = &SelectorExpr{
								X:    &Name{Value: parts[0], expr: posEx},
								Sel:  &Name{Value: parts[1], expr: posEx},
								expr: posEx,
							}
						} else {
							reexpr = &Name{Value: t, expr: posEx}
						}
						list.ElemList = append(list.ElemList, &CompositeLit{
							Type:   reexpr,
							Rbrace: posEx.pos,
							expr:   posEx,
						})
					}
				}
			}
		}
		bodyStmt = &ReturnStmt{
			Results: result,
			stmt:    posSt,
		}
	}

	ifst := &IfStmt{
		Cond: &Operation{
			Op:   Neq,
			X:    name,
			Y:    nilName,
			expr: expr{},
		},
		Then: &BlockStmt{
			List:   []Stmt{bodyStmt},
			Rbrace: ex.Pos(),
			stmt:   posSt,
		},
		stmt: posSt,
	}
	return ifst
}

func NewCallErrorStatement(ex Expr, msg string) Expr {
	exr := expr{node: node{pos: ex.Pos()}}
	return &CallExpr{
		Fun: &SelectorExpr{
			X:    &Name{Value: "errors", expr: exr},
			Sel:  &Name{Value: "New", expr: exr},
			expr: exr,
		},
		ArgList: []Expr{
			&BasicLit{
				Value: fmt.Sprintf(`"%s"`, msg),
				Kind:  StringLit,
				expr:  exr,
			},
		},
		expr: exr,
	}
}

func NewCallErrorStatementFormat(ex Expr) Expr {
	exr := expr{node: node{pos: ex.Pos()}}
	return &CallExpr{
		Fun: &SelectorExpr{
			X:    &Name{Value: "errors", expr: exr},
			Sel:  &Name{Value: "New", expr: exr},
			expr: exr,
		},
		ArgList: []Expr{
			&CallExpr{
				Fun: &SelectorExpr{
					X:    &Name{Value: "fmt", expr: exr},
					Sel:  &Name{Value: "Sprintf", expr: exr},
					expr: exr,
				},
				ArgList: []Expr{
					&BasicLit{
						Value: "To Be Replace",
						Kind:  StringLit,
						expr:  exr,
					},
				},
				expr: exr,
			},
		},
		expr: exr,
	}
}

func NewNilName(ex Expr) *Name {
	return &Name{Value: "nil", expr: expr{node{pos: ex.Pos()}}}
}

func MissingImportDecl() []*ImportDecl {
	return []*ImportDecl{
		{
			Path: &BasicLit{
				Value: `"errors"`,
				Kind:  StringLit,
				expr:  expr{},
			},
			decl: decl{node: node{}},
		},
		{
			Path: &BasicLit{
				Value: `"fmt"`,
				Kind:  StringLit,
				expr:  expr{},
			},
			decl: decl{node: node{}},
		},
	}
}
