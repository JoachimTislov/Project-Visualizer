package ops

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/JoachimTislov/RefViz/internal"
	"github.com/JoachimTislov/RefViz/lsp"
	"github.com/JoachimTislov/RefViz/types"
)

const (
	references = "references"
)

func getRefs(path string, symbol *types.Symbol, refs *map[string]*types.Ref) func() error {
	return func() error {
		pathToSymbol := fmt.Sprintf("%s:%s", path, symbol.Position.String())
		relPath, err := filepath.Rel(internal.ProjectPath(), path)
		if err != nil {
			return fmt.Errorf("error getting relative path: %s, err: %v", path, err)
		}

		log.Printf("\t\t Finding references for symbol: %s\n", symbol.Name)

		output, err := lsp.RunGopls(internal.ProjectPath(), references, pathToSymbol)
		if err != nil {
			cache.LogError(fmt.Sprintf("gopls %s %s", references, pathToSymbol))
			symbol.ZeroRefs = true
			return nil
		}
		// if there are no references, add the symbol to the unused symbols list
		if string(output) == "" {
			symbol.ZeroRefs = true
			// Add to unused map in the cache
			cache.AddUnusedSymbol(relPath, symbol.Name, types.NewUnusedSymbol(
				filepath.Base(filepath.Dir(path)),
				filepath.Base(path),
				pathToSymbol,
			))
		}

		if err := parseRefs(string(output), refs); err != nil {
			return fmt.Errorf("error parsing references: %s, err: %v", pathToSymbol, err)
		}

		return nil
	}
}

func parseRefs(output string, refs *map[string]*types.Ref) error {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		args := strings.Split(line, ":")
		path := args[0]
		LinePos := args[1]

		fileName := filepath.Base(path)
		folderName := filepath.Base(filepath.Dir(path))

		parentSymbolName, err := getRelatedMethod(path, LinePos)
		if err != nil {
			return fmt.Errorf("error getting related method: %s, err: %v", path, err)
		}
		(*refs)[path] = &types.Ref{
			Path:       fmt.Sprintf("%s:%s:%s", path, args[1], args[2]),
			FilePath:   path,
			FolderName: folderName,
			FileName:   fileName,
			MethodName: *parentSymbolName,
		}
	}
	return nil
}

// getRelatedMethod finds the closest method above the reference
func getRelatedMethod(path string, refLinePos string) (*string, error) {
	c, _, err := getSymbols(path, false)
	if err != nil {
		return nil, fmt.Errorf("error getting symbols: %s, err: %v", path, err)
	}
	if len(c.Symbols) == 0 {
		return nil, fmt.Errorf("zero symbols found in %s", path)
	}
	var parentSymbol *types.Symbol
	// loop through potential parent symbols
	for _, s := range c.Symbols {
		// skip if the symbol is not a function
		if s.Kind != function && s.Kind != method {
			continue
		}
		if parentSymbol == nil {
			parentSymbol = s
			continue
		}
		isFurtherDown := parentSymbol.Position.Line < s.Position.Line
		isAboveRef := s.Position.Line < refLinePos
		if isFurtherDown && isAboveRef {
			parentSymbol = s
		}
	}
	if parentSymbol == nil {
		for _, s := range c.Symbols {
			if s.Position.Line == refLinePos {
				return &s.Name, nil
			}
		}
		panic(fmt.Sprintf("Parent symbol is nil, path: %s, line: %s", path, refLinePos))
	}
	return &parentSymbol.Name, nil
}
