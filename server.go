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

	"github.com/gorilla/mux"
)

var addr = flag.String("addr", ":8080",
	"HTTP/REST listen address")
var appsPath = flag.String("appsPath", "./apps",
	"apps directory")
var staticPath = flag.String("staticPath", "./static",
	"path to static web UI content")

func main() {
	flag.Parse()
	log.Printf("%s\n", os.Args[0])
	flag.VisitAll(func(f *flag.Flag) { log.Printf("  -%s=%s\n", f.Name, f.Value) })

	log.Printf("serving: /apps/{app}/")
	start(*addr, *appsPath, *staticPath)
}

func start(addr, appsPath, staticPath string) {
	r := mux.NewRouter()
	sr := r.PathPrefix("/apps/{app}/").Subrouter()
	sr.HandleFunc("/data.json",
		withApp(func(w http.ResponseWriter, r *http.Request, app string) {
			m := map[string]interface{}{}
			err := content(appsPath, ".json", func(path, name string, b []byte) error {
				key := name[0 : len(name)-len(".json")]
				var val interface{}
				err := json.Unmarshal(b, &val)
				if err != nil {
					log.Printf("error: parsing JSON file: %s, err: %v\n", path, err)
					return err
				}
				m[key] = val
				return nil
			})
			if err != nil {
				log.Printf("error: collecting json: %s, err: %v\n", appsPath, err)
				return
			}
			d, err := json.Marshal(m)
			if err != nil {
				log.Printf("error: marshaling json: %s, err: %v\n", appsPath, err)
				return
			}
			w.Write(d)
		}))
	sr.HandleFunc("/data.js",
		suffixHandler(appsPath, ".js", "/* %s.js */", "/* %s.js */"))
	sr.HandleFunc("/data.css",
		suffixHandler(appsPath, ".css", "/* %s.css */", "/* %s.css */"))
	sr.HandleFunc("/data.ract",
		suffixHandler(appsPath, ".ract", "<!-- {{>%s}} -->", "<!-- {{/%s}} -->"))
	sr.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, staticPath+"/index.html")
	})
	r.PathPrefix("/").Handler(http.FileServer(http.Dir(staticPath)))

	http.ListenAndServe(addr, r)
}

// Visits every file in a directory tree that matches a name suffix.
func content(root, suffix string, visitor func(path, name string, b []byte) error) error {
	return filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if f.IsDir() || !strings.HasSuffix(path, suffix) {
			return nil
		}
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		return visitor(path, f.Name(), b)
	})
}

func suffixHandler(root, suffix, beg, end string) func(http.ResponseWriter, *http.Request) {
	return withApp(func(w http.ResponseWriter, r *http.Request, app string) {
		content(root, suffix, func(path, name string, b []byte) error {
			base := name[0 : len(name)-len(suffix)]
			w.Write([]byte(fmt.Sprintf(beg+"\n", base)))
			w.Write(b)
			w.Write([]byte(fmt.Sprintf(end+"\n", base)))
			w.Write([]byte("\n"))
			return nil
		})
	})
}

func withApp(orig func(http.ResponseWriter, *http.Request,
	string)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// TODO: Auth checks.
		orig(w, r, mux.Vars(r)["app"])
	}
}
