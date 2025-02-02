package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Recursive data structure to store the project structure.
// Used for graphviz file generation
type graphMap map[string]*folder

type folder struct {
	FolderName string        `json:"folderName"`
	FolderPath string        `json:"folderPath"`
	Refs       []refGraphviz `json:"refs,omitempty"`
	Files      []file        `json:"files,omitempty"`
	Errors     []goplsError  `json:"errors,omitempty"`
	SubFolders *graphMap     `json:"subFolders,omitempty"`
}

type goplsError struct {
	Error   error  `json:"error,omitempty"`
	Command string `json:"command,omitempty"`
	Input   string `json:"input,omitempty"`
	Output  string `json:"output,omitempty"`
}

type file struct {
	Name    string `json:"name"`
	path    string
	Refs    []refGraphviz    `json:"refs,omitempty"`
	Symbols []symbolGraphviz `json:"symbols,omitempty"`
}

type symbolGraphviz struct {
	Name     string        `json:"name"`
	Kind     string        `json:"kind"`
	Position position      `json:"position"`
	Refs     []refGraphviz `json:"refs,omitempty"`
}

// source is there since the symbol is a reference to a symbol in another file
// Will result in duplicate data, but it's needed to keep track of the source
type refGraphviz struct {
	Source refInfo `json:"source"`
	Info   refInfo `json:"info"`
}

type refInfo struct {
	path       string
	FolderName string `json:"folderName"`
	FileName   string `json:"fileName"`
	MethodName string `json:"methodName"`
}

type position struct {
	Line      string `json:"line"`
	CharRange string `json:"charRange"`
}

func (p position) getPos() string {
	return fmt.Sprintf("%s:%s", p.Line, p.CharRange)
}

type fileMap map[string]struct {
	Path    string     `json:"path"` // relative path to the file
	ModTime int64      `json:"modTime"`
	Symbols *[]*symbol `json:"symbols"`
}

type symbol struct {
	Name     string   `json:"name"`
	Kind     string   `json:"kind"`
	Position position `json:"position"`
	Refs     *[]*ref  `json:"refs,omitempty"`
}

type ref struct {
	Path       string `json:"path"`
	FolderName string `json:"folderName"`
	FileName   string `json:"fileName"`
	MethodName string `json:"methodName"`
}

const (
	function           = "Function"
	method             = "Method"
	cacheFilePath      = "map.json"
	fileMapPath        = "fileMap.json"
	mapFolderPath      = "maps"
	symbols            = "symbols"
	references         = "references"
	both               = "both"
	graphvizFolderPath = "graphviz"
)

type sbMap map[string]bool

var (
	rootFolderName    = "quickfeed"
	rootFolderRelPath = "../../../quickfeed"
	projectMap        = &graphMap{}
	fMap              = &fileMap{}
	errors            []goplsError

	inExt   = sbMap{".go": true, ".ts": true, ".tsx": true} // supported extensions
	exDirs  = sbMap{"node_modules": true, "doc": true, ".git": true}
	exFiles = sbMap{}
)

/*
TODO: Make the error handling return the gopls log instead of the error status message..
TODO: implement libraries which finds references for typescript and react .tsx and .ts files
*/

func main() {
	graphviz := flag.String("graphviz", "", "generate graphviz file with the given map")
	list := flag.Bool("list", false, "list all maps")
	create := flag.String("create", "", "create map")
	delete := flag.String("delete", "", "delete map")
	scan := flag.Bool("scan", false, "scan the project for symbols")
	references := flag.Bool("references", false, "when scanning, also find references")
	content := flag.String("content", "", "name of file or folder to scan, default is everything")
	flag.Parse()
	if *scan {
		scanAll := *content == ""
		if scanAll {
			*content = rootFolderName
		}
		var isDir bool
		if err := handleContentInput(&isDir, scanAll, content); err != nil {
			fmt.Printf("Error handling content input: %v\n", err)
			return
		}
		if err := scanContent(&isDir, references, content); err != nil {
			fmt.Printf("Error scanning content: %v\n", err)
			return
		}
	}
	if *list {
		maps, err := os.ReadDir(mapFolderPath)
		if err != nil {
			fmt.Printf("Error reading directory: %v\n", err)
			return
		}
		for _, m := range maps {
			fmt.Println(m.Name())
		}
		return
	}
	if *create != "" {
		if err := createMap(create); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Map created")
		return
	}
	if *delete != "" {
		if err := deleteMap(delete); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Map deleted")
		return
	}
	if *graphviz != "" {
		// Following can be written with any graphing library
		// Currently, the graph is visualized with graphviz
		// Extension: tintinweb.graphviz-interactive-preview, can display the graph in vscode
		if err := createGraphvizFile(graphviz); err != nil {
			fmt.Println(err)
			return
		}
	}
}

func createMap(name *string) error {
	if _, err := os.Create(getMapPath(name)); err != nil {
		return fmt.Errorf("Error creating map: %v", err)
	}
	return nil
}

func getMapPath(name *string) string {
	return filepath.Join(mapFolderPath, fmt.Sprintf("%s.json", *name))
}

func deleteMap(name *string) error {
	if err := os.Remove(getMapPath(name)); err != nil {
		return fmt.Errorf("Error deleting map: %v", err)
	}
	return nil
}

func scanContent(isDir *bool, scanForRefs *bool, content *string) error {
	// render file map
	if err := getCache(fileMapPath, &fMap); err != nil {
		return fmt.Errorf("Error getting symbols from cache: %s, err: %v", fileMapPath, err)
	}
	if *isDir {
		if err := filepath.WalkDir(*content, func(path string, d os.DirEntry, err error) error {
			if err := isValid(d.IsDir(), &path); err != nil {
				return fmt.Errorf("Error: %s is not a valid entity, err: %v", path, err)
			}
			if !d.IsDir() {
				if err := getContent(&path, scanForRefs); err != nil {
					return fmt.Errorf("Error getting content: %s, err: %v", path, err)
				}
			}
			return nil
		}); err != nil {
			return fmt.Errorf("Error walking through directory: %s, err: %v", *content, err)
		}
	} else {
		if err := getContent(content, scanForRefs); err != nil {
			return fmt.Errorf("Error getting content: %s, err: %v", *content, err)
		}
	}
	return nil
}

func getContent(content *string, scanForRefs *bool) error {
	absPath, err := filepath.Abs(*content)
	if err != nil {
		return fmt.Errorf("Error getting absolute path of file: %s, err: %v", *content, err)
	}
	var symbols []*symbol
	if err := getSymbols(absPath, &symbols); err != nil {
		return fmt.Errorf("Error getting symbols: %s, err: %v", *content, err)
	}
	if *scanForRefs {
		for _, s := range symbols {
			if err := getRefs(absPath, s.Position.getPos(), s.Refs); err != nil {
				return fmt.Errorf("Error getting references: %s, err: %v", *content, err)
			}
		}
	}
	if err := addSymbolsToFile(&symbols, &absPath); err != nil {
		return fmt.Errorf("Error adding symbols to file: %s, err: %v", *content, err)
	}
	return nil
}

func addSymbolsToFile(symbols *[]*symbol, content *string) error {
	f, err := os.Stat(*content)
	if err != nil {
		return fmt.Errorf("Error analyzing content: %s, err: %v", *content, err)
	}
	name := f.Name()
	if entry, ok := (*fMap)[name]; ok {
		if entry.Path, err = getRelPath(*content); err != nil {
			return err
		}
		entry.ModTime = f.ModTime().Unix()
		entry.Symbols = symbols
		(*fMap)[name] = entry
	} else {
		return fmt.Errorf("Error: %s not found in file map", name)
	}
	if err := marshalAndWriteToFile(fMap, fileMapPath); err != nil {
		return err
	}
	return nil
}

func getRelPath(filePath string) (string, error) {
	relPath, err := filepath.Rel(rootFolderRelPath, filePath)
	if err != nil {
		return "", fmt.Errorf("Error getting relative path of file: %s, err: %v", filePath, err)
	}
	return relPath, nil
}

func handleContentInput(isDir *bool, inputIsEmpty bool, content *string) error {
	if inputIsEmpty {
		return nil
	}
	entity, err := os.Stat(*content)
	*isDir = entity.IsDir()
	if err != nil {
		return fmt.Errorf("Error analyzing content: %s, err: %v", *content, err)
	}
	if err := isValid(*isDir, content); err != nil {
		return fmt.Errorf("Error: %s is not a valid entity, err: %v", *content, err)
	}
	return nil
}

func getCache(filePath string, v any) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		os.WriteFile(filePath, []byte("{}"), 0o644)
	}
	if bytes, err := os.ReadFile(filePath); err != nil {
		return fmt.Errorf("Get content from cache error: %s", err)
	} else {
		if err := json.Unmarshal(bytes, &v); err != nil {
			return fmt.Errorf("Unmarshalling error: %s", err)
		}
	}
	return nil
}

func (fMap *graphMap) getCache() error {
	return getCache(cacheFilePath, *fMap)
}

func marshalAndWriteToFile(v any, filePath string) error {
	bytes, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return fmt.Errorf("Error marshalling: %s", err)
	}
	if err := os.WriteFile(filePath, bytes, 0o644); err != nil {
		return fmt.Errorf("Error when writing to file: %s, err: %s", filePath, err)
	}
	return nil
}

// remove entries with zero files and subfolders
func (fMap *graphMap) clean() error {
	for _, key := range getKeysToDelete(fMap) {
		delete(*fMap, key)
	}
	return nil
}

// getKeysToDelete recursively finds all keys with zero files and subfolders
func getKeysToDelete(fMap *graphMap) []string {
	var keysToDelete []string
	for key, folder := range *fMap {
		if len(*folder.SubFolders) == 0 {
			if len(folder.Files) == 0 {
				keysToDelete = append(keysToDelete, key)
			}
		} else {
			getKeysToDelete(folder.SubFolders)
		}
	}
	return keysToDelete
}

// runs the gopls command with the given arguments
func runGopls(args ...string) ([]byte, error) {
	_args := []string{"-vv", "-rpc.trace"}
	return exec.Command("gopls", append(_args, args...)...).Output()
}

func getSymbols(filePath string, s *[]*symbol) error {
	output, err := runGopls(symbols, filePath)
	if err != nil {
		errors = append(errors, goplsError{Error: err, Command: symbols, Input: filePath, Output: string(output)})
		return fmt.Errorf("Error when running gopls command: %s, err: %s", symbols, err)
	}
	extractSymbols(string(output), s)
	return nil
}

// parses the output of the gopls symbols command and extracts the name, kind, and position of each symbol
func extractSymbols(output string, s *[]*symbol) {
	for _, line := range strings.Split(output, "\n") {
		args := strings.Split(line, " ")
		if len(args) < 3 {
			continue
		}
		name := args[0]
		kind := args[1]
		// for methods, remove the receiver type
		if kind == method && strings.Contains(name, ".") {
			name = strings.Split(name, ".")[1]
		}
		*s = append(*s, &symbol{
			Name:     name,
			Kind:     kind,
			Position: createPosition(args[2]),
		})
	}
}

// Gets the line and character range position of the symbol
func createPosition(p string) position {
	args := strings.Split(p, "-")
	args2 := strings.Split(args[0], ":")
	return position{
		Line:      args2[0], // starting line position
		CharRange: fmt.Sprintf("%s-%s", args2[1], strings.Split(args[1], ":")[1]),
	}
}

func getRefs(filePath string, symbolPos string, refs *[]*ref) error {
	pathToSymbol := fmt.Sprintf("%s:%s", filePath, symbolPos)
	output, err := runGopls(references, pathToSymbol)
	if err != nil {
		errors = append(errors, goplsError{Error: err, Command: references, Input: pathToSymbol, Output: string(output)})
		return fmt.Errorf("Error when running gopls command: %s, err: %s", references, err)
	}
	parseRefs(string(output), refs)
	return nil
}

func parseRefs(output string, refs *[]*ref) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		// TODO: Is there a better way ? What library method can be used to parse this?
		filePath := strings.Split(line, ":")[0]
		fileName := getLastEntry(filePath, "/", 0)
		folderName := getLastEntry(filePath, "/", 1)
		ref := &ref{Path: line, FolderName: folderName, FileName: fileName, MethodName: ""}
		*refs = append(*refs, ref)
	}
}

// returns entry relative to last, of a string array with a given delimiter, i determines how many entries from the end
func getLastEntry(str string, delimiter string, i int) string {
	split := strings.Split(str, delimiter)
	return split[len(split)-1-i]
}

// getRelatedMethod finds the closest method above the reference
func getRelatedMethod(symbols []symbol, refParent *symbol, refLinePos string) error {
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

// checks if the directory or file is valid
func isValid(isDir bool, content *string) error {
	if isDir {
		if exDirs[*content] {
			return fmt.Errorf("Error: %s is an excluded directory", *content)
		}
	} else {
		if exFiles[*content] {
			return fmt.Errorf("Error: %s is an excluded file", *content)
		}
		if !inExt[filepath.Ext(*content)] {
			return fmt.Errorf("Error: File is not in a supported extension")
		}
		// bools are initialized to false, so no need to set it to false
	}
	return nil
}

func createGraphvizFile(mapName *string) error {
	// https://golang.org/pkg/text/template/
	// recursive template with nested definitions
	// Whitespace control: https://golang.org/pkg/text/template/#hdr-Text_and_spaces, its a bit tricky
	const tmpl = `
{{- range $folderName, $folder := .Folder}}
digraph {{$folderName}} {
	rankdir=TB;
	{{- template "subgraph" $folder -}}
}
{{- end}}
	{{- define "refs"}}
		{{- $refs := index . 0}}
		{{- $folderName := index . 1}}
		{{- range $ref := $refs}}
			{{$folderName}}_{{trimSpace $ref.Source.MethodName}} -> {{$folderName}}_{{trimSpace $ref.Info.MethodName}};
		{{- end}}
	{{- end}}
{{- define "subgraph"}}
{{- range $folderName, $subfolder := .SubFolders.Folder}}
	subgraph cluster_{{replace $folderName "-" "_"}} {
		label = "{{$folderName}} (folder)";
		rankdir=TB;
		{{- range $file := $subfolder.Files}}
		subgraph cluster_{{replace (replace $file.Name "." "_") "-" "_"}} {
			label = "{{$file.Name}}";
			labelloc="t";
			rankdir=TB;
			{{- range $symbol := $file.Symbols}}
			{{$folderName}}_{{trimSpace $symbol.Name}} [label = "{{trimSpace $symbol.Name}}, {{$symbol.Kind}}";shape = box;];
				{{- template "refs" (arr $symbol.Refs $folderName) -}}
			{{- end}}
		}
		{{- template "refs" (arr $file.Refs $folderName) -}}
		{{- end}}
	}
	{{- template "refs" (arr $subfolder.Refs $folderName) -}}
	{{- template "subgraph" $subfolder -}}
{{- end}}
{{- end}}`
	funcMap := template.FuncMap{
		"replace":   strings.ReplaceAll,
		"trimSpace": strings.TrimSpace,
		"arr": func(els ...any) any { // https://dev.to/moniquelive/passing-multiple-arguments-to-golang-templates-16h8
			return els
		},
	}
	t, err := template.New("graph").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("Error parsing template: %v", err)
	}
	dotFileName := fmt.Sprintf("%s.dot", *mapName)
	dotFilePath := filepath.Join(graphvizFolderPath, dotFileName)
	file, err := os.Create(dotFilePath)
	if err != nil {
		return fmt.Errorf("Error creating dot file: %v", err)
	}
	defer file.Close()
	// Needs to be converted to an graphviz type to be able to use the template
	mapFilePath := getMapPath(mapName)
	customMap := &graphMap{}
	if err := getCache(mapFilePath, customMap); err != nil {
		return fmt.Errorf("Error getting map from cache: %v", err)
	}
	err = t.Execute(file, customMap)
	if err != nil {
		return fmt.Errorf("Error executing template: %v", err)
	}
	return nil
}
