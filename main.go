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
	case "allcallbacks":
		for i, _ := range ns.Callbacks {
			fmt.Println(ns.CallbackToGo(i))
		}
	case "allfunctions":
		for i, _ := range ns.Functions {
			fmt.Println(ns.FunctionToGo(i))
		}
	case "allsignals":
		for i, _ := range ns.Signals {
			fmt.Println(ns.SignalToGo(i))
		}
	case "allvfuncs":
		for i, _ := range ns.VFuncs {
			fmt.Println(ns.VFuncToGo(i))
		}
	case "allconsts":
		for i, _ := range ns.Constants {
			fmt.Println(ns.ConstantToGo(i))
		}
	case "allfields":
		for i, _ := range ns.Fields {
			fmt.Println(ns.FieldToGo(i))
		}
	// TODO properties
	case "allenums":
		for i, _ := range ns.Enums {
			fmt.Println(ns.EnumToGo(i))
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

func (ns Namespace) CallbackToGo(n int) string {
	return ns.Callbacks[n].CallableToGo(ns)
}

func (ns Namespace) FunctionToGo(n int) string {
	return ns.Functions[n].CallableToGo(ns)
}

func (ns Namespace) SignalToGo(n int) string {
	return ns.Signals[n].CallableToGo(ns)
}

func (ns Namespace) VFuncToGo(n int) string {
	return ns.VFuncs[n].CallableToGo(ns)
}

func (cb CallableInfo) CallableToGo(ns Namespace) string {
	if cb.Namespace != ns.Name {
		return "// " + cb.Name + " external; skip"
	}
	s := "func "
	if cb.IsMethod {
		s += "() "
	}
	s += cb.Name + "("
	for _, i := range cb.Args {
		s += ns.ArgToGo(i) + ", "
	}
	s += ")"
	ret := ns.TypeToGo(cb.ReturnType)
	if ret != "" {
		s += " (ret " + ret + ")"
	}
	// TODO return args and errors
	return s
}

func (ns Namespace) ConstantToGo(n int) string {
	c := ns.Constants[n]
	if c.Namespace != ns.Name {
		return "// " + c.Name + " external; skip"
	}
	s := "const " + c.Name + " " + ns.TypeToGo(c.Type) + " = "
	s += "C." + strings.ToUpper(c.Namespace) + "_" + c.Name
	return s
}

func (ns Namespace) FieldToGo(n int) string {
	f := ns.Fields[n]
	if f.Namespace != ns.Name {
		return "\t// " + f.Name + " external; skip"
	}
	s := "\t" + f.Name + " " + ns.TypeToGo(f.Type)
	return s
}

func (ns Namespace) EnumToGo(n int) string {
	e := ns.Enums[n]
	if e.Namespace != ns.Name {
		return "// " + e.Name + " external; skip"
	}
	// TODO use e.Name or e.RTName?
	s := "type " + e.Name + " " + e.StorageType.BasicString() + "\n"
	s += "const (\n"
	for _, i := range e.Values {
		v := ns.Values[i]
		s += "\t" + v.Name + " " + e.Name + " = "
		s += "C." + strings.ToUpper(v.Namespace) + "_" + v.Name + "\n"		
	}
	s += ")"
	return s
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

func (t TypeTag) BasicString() string {
	s := ""
	switch t {
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
	case TagUnichar:
		s += "rune"
	default:
		panic(fmt.Errorf("unknown or non-basic tag type %d", t))
	}
	return s
}
