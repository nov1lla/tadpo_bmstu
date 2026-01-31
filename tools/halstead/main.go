package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type functionMetrics struct {
	File       string  `json:"file"`
	Func       string  `json:"func"`
	Package    string  `json:"package"`
	N1         int     `json:"n1_distinct_operators"`
	N2         int     `json:"n2_distinct_operands"`
	N1Total    int     `json:"N1_total_operators"`
	N2Total    int     `json:"N2_total_operands"`
	Vocabulary int     `json:"vocabulary"`
	Length     int     `json:"length"`
	Volume     float64 `json:"volume"`
	Difficulty float64 `json:"difficulty"`
	Effort     float64 `json:"effort"`
}

type report struct {
	GeneratedAt string            `json:"generated_at"`
	Functions   []functionMetrics `json:"functions"`
	Summary     map[string]any    `json:"summary"`
}

func isGoFile(path string) bool {
	return strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go")
}

func collectGoFiles(roots []string) ([]string, error) {
	var files []string
	seen := map[string]bool{}
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				name := d.Name()
				if name == "vendor" || name == ".git" || name == "node_modules" || name == "benchmark" || name == "test-report" {
					return filepath.SkipDir
				}
				return nil
			}
			if !isGoFile(path) {
				return nil
			}
			abs, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if seen[abs] {
				return nil
			}
			seen[abs] = true
			files = append(files, abs)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(files)
	return files, nil
}

func operatorToken(tok token.Token) bool {
	if tok.IsKeyword() {
		return true
	}
	switch tok {
	case token.ADD, token.SUB, token.MUL, token.QUO, token.REM,
		token.AND, token.OR, token.XOR, token.SHL, token.SHR, token.AND_NOT,
		token.ADD_ASSIGN, token.SUB_ASSIGN, token.MUL_ASSIGN, token.QUO_ASSIGN, token.REM_ASSIGN,
		token.AND_ASSIGN, token.OR_ASSIGN, token.XOR_ASSIGN, token.SHL_ASSIGN, token.SHR_ASSIGN, token.AND_NOT_ASSIGN,
		token.LAND, token.LOR,
		token.ARROW,
		token.INC, token.DEC,
		token.EQL, token.LSS, token.GTR, token.ASSIGN, token.NOT,
		token.NEQ, token.LEQ, token.GEQ, token.DEFINE,
		token.ELLIPSIS,
		token.LPAREN, token.LBRACK, token.LBRACE,
		token.RPAREN, token.RBRACK, token.RBRACE,
		token.COMMA, token.PERIOD, token.SEMICOLON, token.COLON:
		return true
	default:
		return false
	}
}

func operandToken(tok token.Token) bool {
	switch tok {
	case token.IDENT, token.INT, token.FLOAT, token.IMAG, token.CHAR, token.STRING:
		return true
	default:
		return false
	}
}

func halsteadForFunc(filename, pkg, funcName string, src []byte, startOffset, endOffset int) functionMetrics {
	segment := src[startOffset:endOffset]
	fset := token.NewFileSet()
	file := fset.AddFile(filename, -1, len(segment))

	var s scanner.Scanner
	s.Init(file, segment, nil, scanner.ScanComments)

	distOps := map[string]struct{}{}
	distOperands := map[string]struct{}{}
	N1, N2 := 0, 0

	for {
		_, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}
		if operandToken(tok) {
			N2++
			key := lit
			if tok == token.IDENT {
				key = lit
			}
			distOperands[key] = struct{}{}
			continue
		}
		if operatorToken(tok) {
			N1++
			distOps[tok.String()] = struct{}{}
		}
	}

	n1 := len(distOps)
	n2 := len(distOperands)
	vocab := n1 + n2
	length := N1 + N2
	volume := 0.0
	if vocab > 0 && length > 0 {
		volume = float64(length) * math.Log2(float64(vocab))
	}
	difficulty := 0.0
	if n2 > 0 {
		difficulty = (float64(n1) / 2.0) * (float64(N2) / float64(n2))
	}
	effort := difficulty * volume

	return functionMetrics{
		File:       filename,
		Func:       funcName,
		Package:    pkg,
		N1:         n1,
		N2:         n2,
		N1Total:    N1,
		N2Total:    N2,
		Vocabulary: vocab,
		Length:     length,
		Volume:     volume,
		Difficulty: difficulty,
		Effort:     effort,
	}
}

func main() {
	var (
		pathsCSV  = flag.String("paths", "code", "Comma-separated roots to analyze (default: code)")
		outPath   = flag.String("output", "code/product/test-report/static/halstead.json", "Output JSON path")
		maxVolume = flag.Float64("max-volume", 0, "Fail if any function volume is greater than this (0 = do not fail)")
	)
	flag.Parse()

	roots := strings.Split(*pathsCSV, ",")
	files, err := collectGoFiles(roots)
	if err != nil {
		fmt.Fprintln(os.Stderr, "collect files:", err)
		os.Exit(2)
	}

	var functions []functionMetrics
	var volumes []float64

	for _, filePath := range files {
		src, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "read file:", err)
			os.Exit(2)
		}
		fset := token.NewFileSet()
		parsed, err := parser.ParseFile(fset, filePath, src, parser.SkipObjectResolution)
		if err != nil {
			fmt.Fprintln(os.Stderr, "parse file:", err)
			os.Exit(2)
		}
		pkg := parsed.Name.Name

		ast.Inspect(parsed, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				return true
			}
			start := fset.Position(fn.Body.Lbrace).Offset
			end := fset.Position(fn.Body.Rbrace).Offset + 1
			if start < 0 || end <= start || end > len(src) {
				return true
			}
			name := fn.Name.Name
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				name = "method." + name
			}

			m := halsteadForFunc(filePath, pkg, name, src, start, end)
			functions = append(functions, m)
			volumes = append(volumes, m.Volume)
			return true
		})
	}

	sort.Slice(functions, func(i, j int) bool {
		if functions[i].Volume == functions[j].Volume {
			if functions[i].File == functions[j].File {
				return functions[i].Func < functions[j].Func
			}
			return functions[i].File < functions[j].File
		}
		return functions[i].Volume > functions[j].Volume
	})
	sort.Float64s(volumes)

	p95 := 0.0
	if len(volumes) > 0 {
		idx := int(math.Ceil(0.95*float64(len(volumes)))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(volumes) {
			idx = len(volumes) - 1
		}
		p95 = volumes[idx]
	}

	rep := report{
		GeneratedAt: time.Now().Format(time.RFC3339),
		Functions:   functions,
		Summary: map[string]any{
			"functions": len(functions),
			"p95":       p95,
		},
	}

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "mkdir:", err)
		os.Exit(2)
	}
	f, err := os.Create(*outPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create output:", err)
		os.Exit(2)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(rep); err != nil {
		fmt.Fprintln(os.Stderr, "write output:", err)
		os.Exit(2)
	}

	if *maxVolume > 0 {
		for _, fn := range functions {
			if fn.Volume > *maxVolume {
				fmt.Fprintf(os.Stderr, "halstead: volume %.2f exceeds max %.2f (%s:%s)\n", fn.Volume, *maxVolume, fn.File, fn.Func)
				os.Exit(1)
			}
		}
	}
}

