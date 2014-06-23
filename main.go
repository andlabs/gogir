// 23 june 2014
package main

import (
	"fmt"
	"os"
	"encoding/json"
	"io"
	"bytes"
	"sort"
	"strings"
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

func jsonout(w io.Writer, data interface{}) {
	e := json.NewEncoder(w)
	err := e.Encode(data)
	if err != nil { panic(err) }
}

func main() {
	if len(os.Args) != 4 { panic("usage: " + os.Args[0] + " repo ver {json|jsoni}") }
	ns, err := ReadNamespace(os.Args[1], os.Args[2])
	if err != nil { panic(err) }
	switch os.Args[3] {
	case "json":
		jsonout(os.Stdout, ns)
	case "jsoni":
		jsonout(&indenter{os.Stdout}, ns)
	case "innerobj":
		objs := make([]ObjectInfo, len(ns.Objects))
		copy(objs, ns.Objects)
		sort.Sort(sort.Reverse(sort.IntSlice(ns.TopLevelObjects)))		// TODO should we do this ourselves? (minus the reversing)
		for _, i := range ns.TopLevelObjects {
			objs = append(objs[:i], objs[i + 1:]...)
		}
		jsonout(&indenter{os.Stdout}, objs)
	case "allargs":
		for i, _ := range ns.Args {
			fmt.Println(ns.ArgToGo(i))
		}
	default:
		os.Args = os.Args[:1]		// quick hack
		main()
	}
}

func (ns Namespace) ArgToGo(n int) string {
	arg := ns.Args[n]
	return fmt.Sprintf("%s %s", arg.Name, ns.TypeToGo(arg.Type))
}

func (ns Namespace) TypeToGo(n int) string {
	t := ns.Types[n]
	s := ""
	if t.IsPointer {
		switch t.Tag {
		case TagUTF8String, TagFilename, TagArray, TagGList, TagGSList, TagGHashTable:
			// don't add a pointer to these C types
		default:
			s += "*"
		}
	}
	// don't add t.Namespace; that'll produce weird things for cross-included data like gobject.string
	switch t.Tag {
	case TagVoid:
		if t.IsPointer {
			s = "interface{}"
		}
		// otherwise it's a function return; do nothing
	case TagBoolean:
		s += "bool"
	case TagInt8:
		s += "int8"
	case TagUint8:
		s += "uint8"
	case TagInt16:
		s += "int16"
	case TagUint16:
		s += "uint16"
	case TagInt32:
		s += "int32"
	case TagUint32:
		s += "uint32"
	case TagInt64:
		s += "int64"
	case TagUint64:
		s += "uint64"
	case TagFloat:
		s += "float32"
	case TagDouble:
		s += "float64"
	case TagGType:
		s += "GType"
	case TagUTF8String:
		s += "string"
	case TagFilename:
		s += "string"
	case TagArray:
		switch t.ArrayType {
		case CArray, GArray:
			s += "[]"
			s += ns.TypeToGo(t.ParamTypes[0])
		case GPtrArray:
			s += "[]*"
			s += ns.TypeToGo(t.ParamTypes[0])
		case GByteArray:
			s += "[]byte"
		default:
			panic(fmt.Errorf("unknown array type %d", t.ArrayType))
		}
	case TagInterface:
		if t.Interface.Namespace != ns.Name {
			s += strings.ToLower(t.Interface.Namespace) + "."
		}
		s += t.Interface.Name
	case TagGList:
		s += "[]"
		if ns.Types[t.ParamTypes[0]].Tag.GContainerStorePointer() {
			s += "*"
		}
		s += ns.TypeToGo(t.ParamTypes[0])
	case TagGSList:
		s += "[]"
		if ns.Types[t.ParamTypes[0]].Tag.GContainerStorePointer() {
			s += "*"
		}
		s += ns.TypeToGo(t.ParamTypes[0])
	case TagGHashTable:
		s += "map["
		if ns.Types[t.ParamTypes[0]].Tag.GContainerStorePointer() {
			s += "*"
		}
		s += ns.TypeToGo(t.ParamTypes[0])
		s += "]"
		if ns.Types[t.ParamTypes[1]].Tag.GContainerStorePointer() {
			s += "*"
		}
		s += ns.TypeToGo(t.ParamTypes[1])
	case TagGError:
		s += "error"
	case TagUnichar:
		s += "rune"
	default:
		panic(fmt.Errorf("unknown tag type %d", t.Tag))
	}
	return s
}

// for GList, GSList, and GHashTable, whether the stored type is a pointer is not stored; use this function to find out
func (t TypeTag) GContainerStorePointer() bool {
	return t == TagInterface
}
