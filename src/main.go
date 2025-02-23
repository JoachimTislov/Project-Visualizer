package main

import (
	"flag"
	"log"
	"time"

	"github.com/JoachimTislov/RefViz/mappers"
	"github.com/JoachimTislov/RefViz/ops"
)

/*
TODO: implement libraries which finds references for typescript
*/

func init() {
	if err := ops.LoadDefs(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	graphviz := flag.String("graphviz", "", "generate graphviz file with the given map")
	list := flag.Bool("list", false, "list all maps")
	create := flag.String("c", "", "create map")
	delete := flag.String("d", "", "delete map")
	scan := flag.Bool("scan", false, "scan the project for content")
	content := flag.String("content", "", "content to scan, file or folder")
	scanAgain := flag.Bool("again", false, "scan the project for content again")
	flag.Parse()

	checkOps(list, create, delete)
	if *scan {
		startTime, err := ops.Scan(content, scanAgain)
		if err != nil {
			log.Fatalf("Error scanning content: %v\n", err)
		}
		log.Printf("Scanning took: %v\n", time.Since(*startTime))
	}
	if *graphviz != "" {
		// Following can be written with any graphing library
		// Currently, the graph is visualized with graphviz
		// Extension: tintinweb.graphviz-interactive-preview, can display the graph in vscode
		if err := mappers.CreateGraphvizFile(graphviz); err != nil {
			log.Fatalf("error creating graphviz map: %v", err)
		}
	}
}

func checkOps(list *bool, create, delete *string) {
	operations := []struct {
		Condition bool
		Action    func() error
		Msg       string
	}{
		{*list, ops.ListMaps, "error listing maps"},
		{*create != "", func() error { return ops.CreateMap(create) }, "error creating map"},
		{*delete != "", func() error { return ops.DeleteMap(delete) }, "error deleting map"},
	}
	for _, op := range operations {
		if op.Condition {
			if err := op.Action(); err != nil {
				log.Fatalf("%s: %v\n", op.Msg, err)
			}
		}
	}
}
