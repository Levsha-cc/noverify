package langsrv

import (
	"fmt"

	"github.com/Levsha-cc/noverify/src/lintdebug"
	"github.com/Levsha-cc/noverify/src/meta"
	"github.com/Levsha-cc/noverify/src/solver"
	"github.com/Levsha-cc/noverify/src/state"
	"github.com/Levsha-cc/noverify/src/vscode"
	"github.com/z7zmey/php-parser/node"
	"github.com/z7zmey/php-parser/node/expr"
	"github.com/z7zmey/php-parser/node/name"
	"github.com/z7zmey/php-parser/walker"
)

type definitionWalker struct {
	dummyWalker

	st meta.ClassParseState

	position int
	scopes   map[node.Node]*meta.Scope

	result      []vscode.Location
	foundScopes []*meta.Scope
}

func safeExprType(sc *meta.Scope, cs *meta.ClassParseState, n node.Node) (res *meta.TypesMap) {
	defer func() {
		if r := recover(); r != nil {
			res = meta.NewTypesMap(fmt.Sprintf("Panic: %s", fmt.Sprint(r)))
		}
	}()

	res = solver.ExprType(sc, cs, n)
	return
}

// EnterNode is invoked at every node in hierarchy
func (d *definitionWalker) EnterNode(w walker.Walkable) bool {
	n := w.(node.Node)

	sc, ok := d.scopes[n]
	if ok {
		d.foundScopes = append(d.foundScopes, sc)
	}

	state.EnterNode(&d.st, n)

	switch n := w.(type) {
	case *expr.FunctionCall:
		pos := n.Function.GetPosition()

		if d.position > pos.EndPos || d.position < pos.StartPos {
			return true
		}

		var fun meta.FuncInfo
		var ok bool
		var nameStr string

		switch nm := n.Function.(type) {
		case *name.Name:
			nameStr = meta.NameToString(nm)
			fun, ok = meta.Info.GetFunction(d.st.Namespace + `\` + nameStr)
			if !ok && d.st.Namespace != "" {
				fun, ok = meta.Info.GetFunction(`\` + nameStr)
			}
		case *name.FullyQualified:
			nameStr = meta.FullyQualifiedToString(nm)
			fun, ok = meta.Info.GetFunction(nameStr)
		}

		if ok {
			d.result = append(d.result, vscode.Location{
				URI: "file://" + fun.Pos.Filename,
				Range: vscode.Range{
					Start: vscode.Position{Line: int(fun.Pos.Line) - 1},
					End:   vscode.Position{Line: int(fun.Pos.Line) - 1},
				},
			})
		}

		lintdebug.Send("Found function %s: %s:%d", nameStr, fun.Pos.Filename, fun.Pos.Line)
	case *expr.StaticCall:
		pos := n.Call.GetPosition()

		if d.position > pos.EndPos || d.position < pos.StartPos {
			return true
		}

		lintdebug.Send("Static call found")

		// not going to resolve $obj->$someMethod(); calls
		id, ok := n.Call.(*node.Identifier)
		if !ok {
			lintdebug.Send("Static Call is not identifier")
			return true
		}

		className, ok := solver.GetClassName(&d.st, n.Class)
		if !ok {
			return true
		}

		fun, _, ok := solver.FindMethod(className, id.Value)
		if ok {
			d.result = append(d.result, vscode.Location{
				URI: "file://" + fun.Pos.Filename,
				Range: vscode.Range{
					Start: vscode.Position{Line: int(fun.Pos.Line) - 1},
					End:   vscode.Position{Line: int(fun.Pos.Line) - 1},
				},
			})
		}
	case *expr.MethodCall:
		pos := n.Method.GetPosition()

		if d.position > pos.EndPos || d.position < pos.StartPos {
			return true
		}

		lintdebug.Send("Method call found")

		if len(d.foundScopes) == 0 {
			lintdebug.Send("No scope found")
			return true
		}

		foundScope := d.foundScopes[len(d.foundScopes)-1]

		// not going to resolve $obj->$someMethod(); calls
		id, ok := n.Method.(*node.Identifier)
		if !ok {
			lintdebug.Send("Method is not identifier")
			return true
		}

		types := safeExprType(foundScope, &d.st, n.Variable)

		types.Iterate(func(t string) {
			fun, _, ok := solver.FindMethod(t, id.Value)
			if !ok {
				lintdebug.Send("Could not find method for %s::%s", t, id.Value)
				return
			}

			d.result = append(d.result, vscode.Location{
				URI: "file://" + fun.Pos.Filename,
				Range: vscode.Range{
					Start: vscode.Position{Line: int(fun.Pos.Line) - 1},
					End:   vscode.Position{Line: int(fun.Pos.Line) - 1},
				},
			})
		})
	case *expr.PropertyFetch:
		pos := n.Property.GetPosition()

		if d.position > pos.EndPos || d.position < pos.StartPos {
			return true
		}

		lintdebug.Send("Property found")

		if len(d.foundScopes) == 0 {
			lintdebug.Send("No scope found")
			return true
		}

		foundScope := d.foundScopes[len(d.foundScopes)-1]

		// not going to resolve $obj->$someProperty
		id, ok := n.Property.(*node.Identifier)
		if !ok {
			lintdebug.Send("Method is not identifier")
			return true
		}

		types := safeExprType(foundScope, &d.st, n.Variable)

		types.Iterate(func(t string) {
			prop, _, ok := solver.FindProperty(t, id.Value)
			if !ok {
				lintdebug.Send("Could not find property for %s->%s", t, id.Value)
				return
			}

			d.result = append(d.result, vscode.Location{
				URI: "file://" + prop.Pos.Filename,
				Range: vscode.Range{
					Start: vscode.Position{Line: int(prop.Pos.Line) - 1},
					End:   vscode.Position{Line: int(prop.Pos.Line) - 1},
				},
			})
		})
	case *expr.ConstFetch:
		pos := n.Constant.GetPosition()

		if d.position > pos.EndPos || d.position < pos.StartPos {
			return true
		}

		_, c, ok := solver.GetConstant(&d.st, n.Constant)

		if ok {
			d.result = append(d.result, vscode.Location{
				URI: "file://" + c.Pos.Filename,
				Range: vscode.Range{
					Start: vscode.Position{Line: int(c.Pos.Line) - 1},
					End:   vscode.Position{Line: int(c.Pos.Line) - 1},
				},
			})
		}
	case *expr.ClassConstFetch:
		if pos := n.ConstantName.GetPosition(); d.position > pos.EndPos || d.position < pos.StartPos {
			return true
		}

		constName, ok := n.ConstantName.(*node.Identifier)
		if !ok {
			return false
		}

		if constName.Value == `class` || constName.Value == `CLASS` {
			return false
		}

		className, ok := solver.GetClassName(&d.st, n.Class)
		if !ok {
			return false
		}

		if c, _, ok := solver.FindConstant(className, constName.Value); ok {
			d.result = append(d.result, vscode.Location{
				URI: "file://" + c.Pos.Filename,
				Range: vscode.Range{
					Start: vscode.Position{Line: int(c.Pos.Line) - 1},
					End:   vscode.Position{Line: int(c.Pos.Line) - 1},
				},
			})
		}

	case *name.Name:
		pos := n.GetPosition()

		if d.position > pos.EndPos || d.position < pos.StartPos {
			return true
		}

		className, ok := solver.GetClassName(&d.st, n)
		if !ok {
			return true
		}

		c, ok := meta.Info.GetClassOrTrait(className)

		if !ok {
			return true
		}

		d.result = append(d.result, vscode.Location{
			URI: "file://" + c.Pos.Filename,
			Range: vscode.Range{
				Start: vscode.Position{Line: int(c.Pos.Line) - 1},
				End:   vscode.Position{Line: int(c.Pos.Line) - 1},
			},
		})
	case *name.FullyQualified:
		pos := n.GetPosition()
		if d.position > pos.EndPos || d.position < pos.StartPos {
			return true
		}

		className, ok := solver.GetClassName(&d.st, n)
		if !ok {
			return true
		}

		c, ok := meta.Info.GetClassOrTrait(className)

		if !ok {
			return true
		}

		d.result = append(d.result, vscode.Location{
			URI: "file://" + c.Pos.Filename,
			Range: vscode.Range{
				Start: vscode.Position{Line: int(c.Pos.Line) - 1},
				End:   vscode.Position{Line: int(c.Pos.Line) - 1},
			},
		})
	}

	return true
}

// GetChildrenVisitor is invoked at every node parameter that contains children nodes
func (d *definitionWalker) GetChildrenVisitor(key string) walker.Visitor {
	return d
}

// LeaveNode is invoked after node process
func (d *definitionWalker) LeaveNode(w walker.Walkable) {
	n := w.(node.Node)

	if d.scopes != nil {
		_, ok := d.scopes[n]
		if ok && len(d.foundScopes) > 0 {
			d.foundScopes = d.foundScopes[0 : len(d.foundScopes)-1]
		}
	}

	state.LeaveNode(&d.st, n)
}
