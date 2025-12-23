package tempilegocompiler

import (
	"strings"
	"testing"

	tempilecore "github.com/gokhanaltun/tempile-core"
)

func TestCompileSimpleElement(t *testing.T) {
	src := `<div>Hello World</div>`

	options := &CompileOptions{
		PackageName:  "main",
		TemplateName: "Render",
		FileName:     "simple.html",
		SrcPath:      "./",
	}

	code, err := Compile(src, options)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(code, "<div>") || !strings.Contains(code, "Hello World") || !strings.Contains(code, "</div>") {
		t.Fatalf("compiled code missing expected HTML, got:\n%s", code)
	}
}

func TestParseTextNode(t *testing.T) {
	node := &tempilecore.TextNode{Data: "Hello"}
	chunk := parseTextNode(node)
	if !chunk.Writable || chunk.Data != "Hello" {
		t.Fatalf("expected Writable true and Data 'Hello', got %+v", chunk)
	}
}

func TestParseExprNode(t *testing.T) {
	node := &tempilecore.ExprNode{Expr: "data.Name"}
	chunk := parseExprNode(node, &compileContext{})
	expected := "html.EscapeString(fmt.Sprint(data.Name))"
	if chunk.Data != expected {
		t.Fatalf("expected %q, got %q", expected, chunk.Data)
	}
}

func TestParseRawExprNode(t *testing.T) {
	node := &tempilecore.RawExprNode{Expr: "fmt.Println(\"x\")"}
	chunk := parseRawExprNode(node)
	if !chunk.Writable || chunk.Data != "fmt.Println(\"x\")" {
		t.Fatalf("unexpected RawExprNode chunk: %+v", chunk)
	}
}

func TestParseRawCodeNode(t *testing.T) {
	node := &tempilecore.RawCodeNode{Lang: "go", Code: "var x = 5"}
	chunk := parseRawCodeNode(node)
	if chunk.Data != "var x = 5\n" {
		t.Fatalf("unexpected RawCodeNode chunk: %+v", chunk)
	}
}

func TestMergeWritableChunks(t *testing.T) {
	chunks := []*codeChunk{
		{Writable: true, Data: "a"},
		{Writable: true, Data: "b"},
		{Writable: false, Data: "X"},
		{Writable: true, Data: "c"},
	}
	merged := mergeWritableChunks(chunks)
	if len(merged) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(merged))
	}
	if merged[0].Data != "ab" || merged[1].Data != "X" || merged[2].Data != "c" {
		t.Fatalf("merged data mismatch: %+v", merged)
	}
}

func TestParseIfNode(t *testing.T) {
	ifNode := &tempilecore.IfNode{
		Conds: []*tempilecore.Attribute{
			{Name: "go-cond", Value: "data.Flag"},
		},
		Then: []tempilecore.Node{
			&tempilecore.TextNode{Data: "Yes"},
		},
		ElseIfNodes: []*tempilecore.ElseIfNode{
			{
				Conds: []*tempilecore.Attribute{{Name: "go-cond", Value: "data.OtherFlag"}},
				Childs: []tempilecore.Node{
					&tempilecore.TextNode{Data: "Maybe"},
				},
			},
		},
		Else: &tempilecore.ElseNode{
			Childs: []tempilecore.Node{
				&tempilecore.TextNode{Data: "No"},
			},
		},
		Pos: tempilecore.Pos{FileName: "test.html", Line: 1},
	}

	chunks, err := parseIfNode(ifNode, &compileContext{})
	if err != nil {
		t.Fatal(err)
	}

	code := ""
	for _, c := range chunks {
		if c.Writable {
			code += c.Data
		} else {
			code += c.Data
		}
	}

	if !strings.Contains(code, "if data.Flag {") ||
		!strings.Contains(code, "else if data.OtherFlag {") ||
		!strings.Contains(code, "else {") {
		t.Fatalf("compiled if/elseif/else code missing expected parts:\n%s", code)
	}
}

func TestParseForNode(t *testing.T) {
	forNode := &tempilecore.ForNode{
		Loops: []*tempilecore.Attribute{
			{Name: "go-loop", Value: "i, v := range data.Items"},
		},
		Childs: []tempilecore.Node{
			&tempilecore.TextNode{Data: "Item"},
		},
		Pos: tempilecore.Pos{FileName: "test.html", Line: 1},
	}

	chunks, err := parseForNode(forNode, &compileContext{})
	if err != nil {
		t.Fatal(err)
	}

	code := ""
	for _, c := range chunks {
		if c.Writable {
			code += c.Data
		} else {
			code += c.Data
		}
	}

	if !strings.Contains(code, "for i, v := range data.Items {") ||
		!strings.Contains(code, "Item") ||
		!strings.Contains(code, "}") {
		t.Fatalf("compiled for loop code missing expected parts:\n%s", code)
	}
}

func TestParseChildNodes(t *testing.T) {
	nodes := []tempilecore.Node{
		&tempilecore.TextNode{Data: "Hello"},
		&tempilecore.ExprNode{Expr: "data.Name"},
	}

	chunks, err := parseChildNodes(nodes, &compileContext{})
	if err != nil {
		t.Fatal(err)
	}

	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	if chunks[0].Data != "Hello" || !chunks[0].Writable {
		t.Errorf("first chunk incorrect: %+v", chunks[0])
	}

	expected := "html.EscapeString(fmt.Sprint(data.Name))"
	if chunks[1].Data != expected || !chunks[1].Writable || !chunks[1].NoMerge {
		t.Errorf("second chunk incorrect: %+v", chunks[1])
	}
}

func TestParseElementNodeWithAttrAndExpr(t *testing.T) {
	node := &tempilecore.ElementNode{
		Tag: "div",
		Attrs: []*tempilecore.Attribute{
			{
				Name:  "class",
				Value: "container{{data.Class}}",
				ValueNodes: []tempilecore.Node{
					&tempilecore.TextNode{Data: "container"},
					&tempilecore.ExprNode{Expr: "data.Class"},
				},
			},
		},
		Childs: []tempilecore.Node{
			&tempilecore.TextNode{Data: "Hello"},
			&tempilecore.ExprNode{Expr: "data.Name"},
		},
	}

	chunks, err := parseElementNode(node, &compileContext{})
	if err != nil {
		t.Fatal(err)
	}

	// compiler çıktısını tek stringde topluyoruz
	code := ""
	for _, c := range chunks {
		code += c.Data
	}

	// Her parçayı ayrı ayrı kontrol ediyoruz
	expectedParts := []string{
		`<div class="`,
		`container`,
		`html.EscapeString(fmt.Sprint(data.Class))`,
		`">Hello`,
		`html.EscapeString(fmt.Sprint(data.Name))`,
		`</div>`,
	}

	for _, part := range expectedParts {
		if !strings.Contains(code, part) {
			t.Fatalf("compiled element code missing expected part %q:\n%s", part, code)
		}
	}
}
