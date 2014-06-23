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

func main() {
	e := json.NewEncoder(&indenter{os.Stdout})
	ns, err := ReadNamespace(os.Args[1], os.Args[2])
	if err != nil { panic(err) }
	err = e.Encode(ns)
	if err != nil { panic(err) }
}
