package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var addr = flag.String("addr", ":8080",
	"HTTP/REST listen address")
var dataPath = flag.String("dataPath", "./data",
	"data directory")
var dataSuffix = flag.String("dataSuffix", ".json",
	"suffix for data files")
var staticPath = flag.String("staticPath", "./static",
	"path to static web UI content")

func main() {
	flag.Parse()
	fmt.Printf("%s\n", os.Args[0])
	flag.VisitAll(func(f *flag.Flag) {
		fmt.Printf("  -%s=%s\n", f.Name, f.Value)
	})

	dataHandler := func(w http.ResponseWriter, r *http.Request) {
		d, err := json.Marshal(content(*dataPath, *dataSuffix, map[string]interface{}{}))
		if err != nil {
			log.Printf("error: marshaling content: %s, err: %v\n", *dataPath, err)
			return
		}
		w.Write(d)
	}

	http.HandleFunc("/data.json", dataHandler)

	http.Handle("/", http.FileServer(http.Dir(*staticPath)))

	http.ListenAndServe(*addr, nil)
}

// Reads all the json files in a directory tree into a map.
func content(root, suffix string, res map[string]interface{}) map[string]interface{} {
	filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() || !strings.HasSuffix(path, suffix) {
			return nil
		}
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		key := f.Name()[0:len(f.Name())-len(suffix)]
		var val interface{}
		err = json.Unmarshal(b, &val)
		if err != nil {
			log.Printf("error: parsing file: %s, err: %v\n", path, err)
			return err
		}
		res[key] = val
		return nil
	})
	return res
}
