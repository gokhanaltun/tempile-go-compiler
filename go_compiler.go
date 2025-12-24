package tempilegocompiler

import (
	"errors"
	"fmt"
	"go/format"
	"slices"
	"strings"

	tempilecore "github.com/gokhanaltun/tempile-core"
)

type CompileOptions struct {
	PackageName  string
	TemplateName string
	FileName     string
	SrcPath      string
}

type codeChunk struct {
	Writable bool
	NoMerge  bool
	Data     string
}

type compileContext struct {
	usedHTML bool
	usedFMT  bool
}

type importCtx struct {
	imports []string
}

func (i *importCtx) add(imports []string) {
	for _, imp := range imports {
		if !slices.Contains(i.imports, imp) {
			i.imports = append(i.imports, imp)
		}
	}
}

var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true, "embed": true,
	"hr": true, "img": true, "input": true, "link": true, "meta": true,
	"param": true, "source": true, "track": true, "wbr": true,
}

func Compile(src string, options *CompileOptions) (string, error) {
	if options == nil {
		return "", errors.New("missing compile options")
	}

	if options.PackageName == "" {
		return "", errors.New("missing package name in compile options")
	}

	if options.TemplateName == "" {
		return "", errors.New("missing template name in compile options")
	}

	if options.FileName == "" {
		return "", errors.New("missing file name in compile options")
	}

	if options.SrcPath == "" {
		return "", errors.New("missing src path in compile options")
	}

	layout := `
	package %s
	import (
		"io"
		%s
	)

	func %s(w io.Writer, data map[string]any) error {
		var err error
		
		%s

		return err
	}
	`

	ast, err := tempilecore.Parse(src, options.FileName)
	if err != nil {
		return "", err
	}

	ast.ResolveIncludes(options.SrcPath)
	ast.MatchSlotsAndContents()

	ctx := &compileContext{}
	impCtx := &importCtx{}

	var codeChunks []*codeChunk
	for _, node := range ast.Childs {
		chunks, err := parseNode(node, ctx, impCtx)
		if err != nil {
			return "", err
		}
		codeChunks = append(codeChunks, chunks...)

	}

	codeChunks = mergeWritableChunks(codeChunks)

	var code strings.Builder
	for _, chunk := range codeChunks {
		if chunk.Writable {
			if chunk.NoMerge {
				writeStringLiteral(&code, chunk.Data)
			} else {
				code.WriteString(fmt.Sprintf("if _, err = io.WriteString(w, `%s`); err != nil { return err }\n", chunk.Data))
			}
		} else {
			code.WriteString(chunk.Data)
		}
	}

	imports := ""
	if ctx.usedHTML {
		imports += "\t\"html\"\n"
	}
	if ctx.usedFMT {
		imports += "\t\"fmt\"\n"
	}

	for _, imp := range impCtx.imports {
		imports += fmt.Sprintf("\t\"%s\"\n", imp)
	}

	compiledCode := fmt.Sprintf(layout, options.PackageName, imports, options.TemplateName, code.String())

	formattedCode, err := format.Source([]byte(compiledCode))
	if err != nil {
		return "", err
	}

	return string(formattedCode), nil
}

func mergeWritableChunks(chunks []*codeChunk) []*codeChunk {
	var merged []*codeChunk
	var buffer string

	flush := func() {
		if buffer == "" {
			return
		}
		merged = append(merged, &codeChunk{
			Writable: true,
			Data:     buffer,
		})
		buffer = ""
	}

	for _, c := range chunks {
		if c == nil {
			continue
		}

		if c.Writable && !c.NoMerge {
			if buffer == "" {
				buffer = c.Data
			} else {
				buffer = buffer + c.Data
			}
			continue
		}

		flush()
		merged = append(merged, c)
	}

	flush()
	return merged
}

func writeStringLiteral(w *strings.Builder, s string) {
	if strings.Contains(s, "`") {
		w.WriteString(fmt.Sprintf(
			"if _, err = io.WriteString(w, %q); err != nil { return err }\n",
			s,
		))
	} else {
		w.WriteString(fmt.Sprintf(
			"if _, err = io.WriteString(w, `%s`); err != nil { return err }\n",
			s,
		))
	}
}

func parseNode(node tempilecore.Node, ctx *compileContext, impCtx *importCtx) ([]*codeChunk, error) {
	switch node.Type() {
	case tempilecore.NodeImport:
		return []*codeChunk{parseImportNode(node, impCtx)}, nil
	case tempilecore.NodeDocumentType:
		return []*codeChunk{parseDocumentTypeNode(node)}, nil
	case tempilecore.NodeComment:
		return []*codeChunk{parseCommentNode(node)}, nil
	case tempilecore.NodeText:
		return []*codeChunk{parseTextNode(node)}, nil
	case tempilecore.NodeElement:
		return parseElementNode(node, ctx, impCtx)
	case tempilecore.NodeIf:
		return parseIfNode(node, ctx, impCtx)
	case tempilecore.NodeFor:
		return parseForNode(node, ctx, impCtx)
	case tempilecore.NodeRawCode:
		return []*codeChunk{parseRawCodeNode(node)}, nil
	case tempilecore.NodeRawExpr:
		return []*codeChunk{parseRawExprNode(node)}, nil
	case tempilecore.NodeExpr:
		return []*codeChunk{parseExprNode(node, ctx)}, nil
	default:
		return nil, nil
	}
}

func parseImportNode(node tempilecore.Node, impCtx *importCtx) *codeChunk {
	importNode := node.(*tempilecore.ImportNode)

	for _, a := range importNode.Attrs {
		if a.Name == "go" {
			impCtx.add([]string{a.Value})
		}
	}
	return nil
}

func parseDocumentTypeNode(node tempilecore.Node) *codeChunk {
	doctypeNode := node.(*tempilecore.DocumentTypeNode)
	return &codeChunk{
		Writable: true,
		Data:     fmt.Sprintf("%s", doctypeNode.Data),
	}
}

func parseCommentNode(node tempilecore.Node) *codeChunk {
	commentNode := node.(*tempilecore.CommentNode)
	return &codeChunk{
		Writable: true,
		Data:     fmt.Sprintf("%s", commentNode.Data),
	}
}

func parseTextNode(node tempilecore.Node) *codeChunk {
	textNode := node.(*tempilecore.TextNode)
	return &codeChunk{
		Writable: true,
		Data:     fmt.Sprintf("%s", textNode.Data),
	}
}

func parseElementNode(node tempilecore.Node, ctx *compileContext, impCtx *importCtx) ([]*codeChunk, error) {
	elementNode := node.(*tempilecore.ElementNode)
	var chunks []*codeChunk

	tag := elementNode.Tag

	if len(elementNode.Attrs) > 0 {
		chunks = append(chunks, &codeChunk{
			Writable: true,
			Data:     fmt.Sprintf("<%s", tag),
		})
		for _, a := range elementNode.Attrs {
			chunks = append(chunks, &codeChunk{
				Writable: true,
				Data:     fmt.Sprintf(" %s=\"", a.Name),
			})

			for _, n := range a.ValueNodes {
				if n.Type() == tempilecore.NodeText {
					chunks = append(chunks, parseTextNode(n))
				} else if n.Type() == tempilecore.NodeExpr {
					chunks = append(chunks, parseExprNode(n, ctx))
				}
			}
			chunks = append(chunks, &codeChunk{
				Writable: true,
				Data:     "\"",
			})
		}
		chunks = append(chunks, &codeChunk{
			Writable: true,
			Data:     ">",
		})
	} else {
		chunks = append(chunks, &codeChunk{
			Writable: true,
			Data:     fmt.Sprintf("<%s>", tag),
		})
	}

	childChunks, err := parseChildNodes(elementNode.Childs, ctx, impCtx)
	if err != nil {
		return nil, err
	}
	chunks = append(chunks, childChunks...)

	if !voidElements[tag] {
		chunks = append(chunks, &codeChunk{
			Writable: true,
			Data:     fmt.Sprintf("</%s>", tag),
		})
	}

	return chunks, nil
}

func parseIfNode(node tempilecore.Node, ctx *compileContext, impCtx *importCtx) ([]*codeChunk, error) {
	ifNode := node.(*tempilecore.IfNode)
	var chunks []*codeChunk

	var cond string
	for _, c := range ifNode.Conds {
		if c.Name == "go-cond" {
			cond = c.Value
			break
		}
	}

	if cond == "" {
		return nil, fmt.Errorf("missing go-cond in \"if\" element. file: %s line: %d col: %d",
			ifNode.Pos.FileName, ifNode.Pos.Line, ifNode.Pos.Column)
	}

	chunks = append(chunks, &codeChunk{
		Writable: false,
		Data:     fmt.Sprintf("if %s {\n", cond),
	})

	childChunks, err := parseChildNodes(ifNode.Then, ctx, impCtx)
	if err != nil {
		return nil, err
	}
	chunks = append(chunks, childChunks...)
	chunks = append(chunks, &codeChunk{
		Writable: false,
		Data:     "}",
	})

	if len(ifNode.ElseIfNodes) > 0 {
		for _, elseif := range ifNode.ElseIfNodes {
			elseifNode, err := parseElseIfNode(elseif, ctx, impCtx)
			if err != nil {
				return nil, err
			}
			chunks = append(chunks, elseifNode...)
		}
	}

	if ifNode.Else != nil {
		elseNodeChunks, err := parseElseNode(ifNode.Else, ctx, impCtx)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, elseNodeChunks...)
	}

	chunks = append(chunks, &codeChunk{
		Writable: false,
		Data:     "\n",
	})
	return chunks, nil
}

func parseElseIfNode(node tempilecore.Node, ctx *compileContext, impCtx *importCtx) ([]*codeChunk, error) {
	elseIfNode := node.(*tempilecore.ElseIfNode)

	var chunks []*codeChunk

	var cond string
	for _, c := range elseIfNode.Conds {
		if c.Name == "go-cond" {
			cond = c.Value
			break
		}
	}

	if cond == "" {
		return nil, fmt.Errorf("missing go-cond in \"elseif\" element. file: %s line: %d col: %d",
			elseIfNode.Pos.FileName, elseIfNode.Pos.Line, elseIfNode.Pos.Column)
	}

	chunks = append(chunks, &codeChunk{
		Writable: false,
		Data:     fmt.Sprintf("else if %s {\n", cond),
	})

	childChunks, err := parseChildNodes(elseIfNode.Childs, ctx, impCtx)
	if err != nil {
		return nil, err
	}
	chunks = append(chunks, childChunks...)
	chunks = append(chunks, &codeChunk{
		Writable: false,
		Data:     "}",
	})

	return chunks, nil
}

func parseElseNode(node tempilecore.Node, ctx *compileContext, impCtx *importCtx) ([]*codeChunk, error) {
	elseNode := node.(*tempilecore.ElseNode)

	var chunks []*codeChunk

	chunks = append(chunks, &codeChunk{
		Writable: false,
		Data:     "else {\n",
	})

	childChunks, err := parseChildNodes(elseNode.Childs, ctx, impCtx)
	if err != nil {
		return nil, err
	}
	chunks = append(chunks, childChunks...)
	chunks = append(chunks, &codeChunk{
		Writable: false,
		Data:     "}",
	})

	return chunks, nil
}

func parseForNode(node tempilecore.Node, ctx *compileContext, impCtx *importCtx) ([]*codeChunk, error) {
	forNode := node.(*tempilecore.ForNode)
	var chunks []*codeChunk

	var loop string
	for _, l := range forNode.Loops {
		if l.Name == "go-loop" {
			loop = l.Value
			break
		}
	}

	if loop == "" {
		return nil, fmt.Errorf("missing go-loop in \"for\" element. file: %s line: %d col: %d",
			forNode.Pos.FileName, forNode.Pos.Line, forNode.Pos.Column)
	}

	chunks = append(chunks, &codeChunk{
		Writable: false,
		Data:     fmt.Sprintf("for %s {\n", loop),
	})

	childChunks, err := parseChildNodes(forNode.Childs, ctx, impCtx)
	if err != nil {
		return nil, err
	}
	chunks = append(chunks, childChunks...)

	chunks = append(chunks, &codeChunk{
		Writable: false,
		Data:     "}\n",
	})

	return chunks, nil
}

func parseChildNodes(childs []tempilecore.Node, ctx *compileContext, impCtx *importCtx) ([]*codeChunk, error) {
	var chunks []*codeChunk
	for _, c := range childs {
		childChunks, err := parseNode(c, ctx, impCtx)
		if err != nil {
			return nil, err
		}
		chunks = append(chunks, childChunks...)
	}
	return chunks, nil
}

func parseExprNode(node tempilecore.Node, ctx *compileContext) *codeChunk {
	exprNode := node.(*tempilecore.ExprNode)

	ctx.usedHTML = true
	ctx.usedFMT = true

	return &codeChunk{
		Writable: true,
		NoMerge:  true,
		Data:     fmt.Sprintf("html.EscapeString(fmt.Sprint(%s))", exprNode.Expr),
	}
}

func parseRawCodeNode(node tempilecore.Node) *codeChunk {
	rawCodeNode := node.(*tempilecore.RawCodeNode)
	if rawCodeNode.Lang == "go" {
		return &codeChunk{
			Writable: false,
			Data:     fmt.Sprintf("%s\n", rawCodeNode.Code),
		}
	}
	return nil
}

func parseRawExprNode(node tempilecore.Node) *codeChunk {
	rawExprNode := node.(*tempilecore.RawExprNode)
	return &codeChunk{
		Writable: true,
		NoMerge:  true,
		Data:     rawExprNode.Expr,
	}
}
