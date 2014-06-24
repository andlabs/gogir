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
	case "allconsts":
		for i, _ := range ns.Constants {
			fmt.Println(ns.ConstantToGo(i))
		}
	case "allfields":
		for i, _ := range ns.Fields {
			fmt.Println(ns.FieldToGo(i))
		}
	// TODO properties
	case "allobjects":
		for i, _ := range ns.Objects {
			fmt.Println(ns.ObjectToGo(i))
		}
	case "allstructs":
		for i, _ := range ns.Structs {
			fmt.Println(ns.StructToGo(i))
		}
	case "allunions":
		for i, _ := range ns.Unions {
			fmt.Println(ns.UnionToGo(i))
		}
	case "alltypes":
		for i, _ := range ns.Types {
			fmt.Println(ns.TypeToGo(i))
		}
	case "badtypes":
		for i, _ := range ns.Types {
			s := ns.TypeToGo(i)
			if s == "" {
				t := ns.Types[i]
				if t.Tag != TagVoid && !t.IsPointer {		// skip void returns
					fmt.Printf("%d %#v\n", i, ns.Types[i])
				}
			}
		}
	case "gen":
		generate(ns)
	default:
		os.Args = os.Args[:1]		// quick hack
		main()
	}
}

func (ns Namespace) ArgToGo(n int) string {
	arg := ns.Args[n]
	return ns.ArgValueToGo(arg, ns.Types[arg.Type], true)
}

func (ns Namespace) ArgValueToGo(arg ArgInfo, argtype TypeInfo, isArg bool) string {
	return fmt.Sprintf("%s %s", arg.Name, ns.TypeValueToGo(argtype, isArg))
}

func (ns Namespace) GoFuncSig(ci CallableInfo) string {
	s := ns.GoName(ci) + "("
//	for _, i := range ci.Args {
//		s += ns.ArgToGo(i) + ", "
//	}
	s += ")"
	ret := ns.TypeToGo(ci.ReturnType)
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
	s := "const " + ns.GoName(c) + " " + ns.TypeToGo(c.Type) + " = "
	s += "C." + ns.CName(c)
	return s
}

func (ns Namespace) FieldToGo(n int) string {
	f := ns.Fields[n]
	if f.Namespace != ns.Name {
		return "\t// " + f.Name + " external; skip"
	}
	s := "\t" + ns.GoName(f) + " " + ns.TypeToGo(f.Type)
	return s
}

func (ns Namespace) ObjectToGo(n int) string {
	o := ns.Objects[n]
	if o.Namespace != ns.Name {
		return "// " + o.Name + " external; skip"
	}
	s := "type " + ns.GoName(o) + " struct {\n"
	if o.Parent != -1 {
		s += "\t" + ns.GoName(ns.Objects[o.Parent]) + "\n"
	}
	s += "\t// interfaces\n"
	for _, n := range o.Interfaces {
		i := ns.Interfaces[n]
		s += "\t" + ns.GoName(i) + "\n"
	}
	s += "\t//fields\n"
	for _, n := range o.Fields {
		s += ns.FieldToGo(n) + "\n"
	}
	s += "}\n"
	s += "// methods\n"
	// TODO Struct
	for _, n := range o.Constants {
		s += ns.ConstantToGo(n) + "\n"
	}
	// TODO the four functions
	return s
}

func (ns Namespace) StructToGo(n int) string {
	st := ns.Structs[n]
	if st.Namespace != ns.Name {
		return "// " + st.Name + " external; skip"
	}
	s := "type " + ns.GoName(st) + " struct {\n"
	if st.IsClassStruct {
		s += "\t// class structure\n"
	}
	for _, n := range st.Fields {
		s += ns.FieldToGo(n) + "\n"
	}
	s += "}\n"
	return s
}

func (ns Namespace) UnionToGo(n int) string {
	u := ns.Unions[n]
	if u.Namespace != ns.Name {
		return "// " + u.Name + " external; skip"
	}
	s := "type " + ns.GoName(u) + " struct {\n"
	s += "\t//union\n"
	for _, n := range u.Fields {
		s += ns.FieldToGo(n) + "\n"
	}
	s += "}\n"
	return s
}

func (ns Namespace) TypeToGo(n int) string {
	return ns.TypeValueToGo(ns.Types[n], false)
}

func (ns Namespace) TypeValueToGo(t TypeInfo, isArg bool) string {
	s := ""
	if t.IsPointer {
		switch t.Tag {
		case TagUTF8String, TagFilename, TagArray, TagGList, TagGSList, TagGHashTable:
			// don't add a pointer to these C types
		case TagInterface:
			// see GContainerStorePointer below
			if t.Interface.Type == TypeInterface {
				break
			}
			if isArg {		// arguments become the equivalent interfaces; see below
				break
			}
			fallthrough
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
		if isArg {		// arguments become the equivalent interfaces
			s += "I"
		}
		s += t.Interface.Name
	case TagGList:
		s += "[]"
		if ns.Types[t.ParamTypes[0]].GContainerStorePointer() {
			s += "*"
		}
		s += ns.TypeToGo(t.ParamTypes[0])
	case TagGSList:
		s += "[]"
		if ns.Types[t.ParamTypes[0]].GContainerStorePointer() {
			s += "*"
		}
		s += ns.TypeToGo(t.ParamTypes[0])
	case TagGHashTable:
		s += "map["
		if ns.Types[t.ParamTypes[0]].GContainerStorePointer() {
			s += "*"
		}
		s += ns.TypeToGo(t.ParamTypes[0])
		s += "]"
		if ns.Types[t.ParamTypes[1]].GContainerStorePointer() {
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
// interfaces become Go interfaces which are /references/, so don't make htem pointers either
func (t TypeInfo) GContainerStorePointer() bool {
	return t.Tag == TagInterface && t.Interface.Type != TypeInterface
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
