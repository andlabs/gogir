// 23 june 2014
package main

import (
//	"fmt"
	"os"
	"encoding/json"
	"io"
	"bytes"
)

type indenter struct {
	w	io.Writer
}
func (i *indenter) Write(p []byte) (n int, err error) {
	b := new(bytes.Buffer)
	err = json.Indent(b, p, "", " ")//"\t")
	if err != nil { return 0, err }
	return i.w.Write(b.Bytes())
}

var ns Namespace

func jsonout(w io.Writer) {
	e := json.NewEncoder(w)
	err := e.Encode(ns)
	if err != nil { panic(err) }
}

func main() {
	var err error

	if len(os.Args) != 4 { panic("usage: " + os.Args[0] + " repo ver {json|jsoni}") }
	ns, err = ReadNamespace(os.Args[1], os.Args[2])
	if err != nil { panic(err) }
	switch os.Args[3] {
	case "json":
		jsonout(os.Stdout)
	case "jsoni":
		jsonout(&indenter{os.Stdout})
	default:
		os.Args = os.Args[:1]		// quick hack
		main()
	}
}
