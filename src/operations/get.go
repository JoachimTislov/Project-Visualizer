package op

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/JoachimTislov/Project-Visualizer/types"
)

// GetFile reads the content of the file and unmarshals it into the given variable
func GetFile(filePath string, v any) error {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("get content from cache error: %s", err)
	}
	if err := json.Unmarshal(bytes, &v); err != nil {
		return fmt.Errorf("unmarshaling error: %s", err)
	}
	return nil
}

func getContent(path string, scanForRefs *bool) error {
	symbols, err := getSymbols(path)
	if err != nil {
		return fmt.Errorf("error getting symbols: %s, err: %v", path, err)
	}
	if *scanForRefs {
		for _, s := range symbols {
			if err := getRefs(path, s.Position.String(), s.Refs); err != nil {
				return fmt.Errorf("error getting references: %s, err: %v", path, err)
			}
		}
	}
	if err := cacheSymbols(symbols, path); err != nil {
		return fmt.Errorf("error caching symbols: %s, err: %v", path, err)
	}
	return nil
}

// getRelatedMethod finds the closest method above the reference
func getRelatedMethod(symbols []types.Symbol, refParent *types.Symbol, refLinePos string) error {
	// loop through potential parent symbols
	for _, s := range symbols {
		// skip if the symbol is not a function
		if s.Kind != function && s.Kind != method {
			continue
		}
		isFurtherDown := refParent.Position.Line < s.Position.Line
		isAboveRef := s.Position.Line < refLinePos
		// if the new method is further down and above the reference, update the refParent
		if isFurtherDown && isAboveRef {
			*refParent = s
		}
	}
	return nil
}
