package httpfilter

import (
	"net/http"
	"path/filepath"
	"mime"
	"io/ioutil"
	"errors"
)

var filterFileName = "_filters.txt"

type Server struct {
	root string
	ops map[string]OpFunc
}

func NewServer(root string, ops ...map[string]OpFunc) * Server {
	sv := &Server{
		root: root,
	}
	sv.ops = map[string]OpFunc{
		"deft":     sv.serveFile,
		"ignore":   ignore,
		"pseudo":   pseudo,
		"redirect": redirect,
	}
	for _, m := range ops {
		for k, v := range m {
			sv.ops[k] = v
		}
	}
	return sv
}

func(sv * Server) ServeHTTP(w http.ResponseWriter, req * http.Request) {
	wr := wrapWriter(w)
	query := filepath.Join(sv.root, req.URL.Path)
	dir := filepath.Dir(query)
	query = filepath.Base(query)
	filters := parseFilterFile(filepath.Join(dir, filterFileName))
	for _, v := range filters {
		if !<-wr.ok {
			break
		}
		wr.ok <- true
		if match(query,v[1]) {
			if op := sv.ops[v[0]]; op != nil {
				query = op(wr, req, query, v[2:])
			} else {
				panic(errors.New("Undefined operator " + v[0]))
			}
		}
	}
}

func(sv * Server) serveFile(w http.ResponseWriter, req * http.Request, query string, args []string) string {
	if query == filterFileName {
		http.Error(w, "Not found.", 404)
		return ""
	}
	name := filepath.Dir(req.URL.Path)
	name = filepath.Join(sv.root, name)
	name = filepath.Join(name, query)
	b, err := ioutil.ReadFile(name)
	if err != nil {
		http.Error(w, "Not found.", 404)
		return ""
	}
	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(query)))
	w.Write(b)
	return ""
}

func match(q, s string) bool {
	se := filepath.Ext(s)		//selector ext
	sn := s[:len(s)-len(se)]	//selector name
	qe := filepath.Ext(q)		//query ext
	qn := q[:len(q)-len(qe)]	//query name
	return 	(q == s) || 				//name and extension match
			(s == "*") || 				//selector is *
			(se == ".*" && qn == sn) || //selector ext is * and name matches
			(sn == "*" && qe == se)		//selector name is * and ext matches
}