// 23 june 2014
package main

import (
	"fmt"
	"os"
	"encoding/json"
	"io"
	"bytes"
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
	namespace = ns.Name
	switch os.Args[3] {
	case "json":
		jsonout(os.Stdout, ns)
	case "jsoni":
		jsonout(&indenter{os.Stdout}, ns)
	case "gen":
		generate(ns)
	default:
		os.Args = os.Args[:1]		// quick hack
		main()
	}
}

func ArgToGo(arg *ArgInfo) string {
	return fmt.Sprintf("%s %s", arg.Name, arg.Type.GoType(true))
}

func GoFuncSig(ci CallableInfo) string {
	s := GoName(ci) + "("
	for _, i := range ci.Args {
		s += ArgToGo(i) + ", "
	}
	s += ")"
	ret := TypeToGo(ci.ReturnType, false)
	if ret != "" {
		s += " (ret " + ret + ")"
	}
	// TODO return args and errors
	return s
}

func ConstantToGo(c *ConstantInfo) string {
	if c.Namespace != namespace {
		return "// " + c.Name + " external; skip"
	}
	s := "const " + GoName(c) + " " + TypeToGo(c.Type, false) + " = "
	s += "C." + CName(c)
	return s
}

func FieldToGo(f *FieldInfo) string {
	if f.Namespace != namespace {
		return "\t// " + f.Name + " external; skip"
	}
	s := "\t" + GoName(f) + " " + TypeToGo(f.Type, false)
	return s
}

func UnionToGo(u *UnionInfo) string {
	if u.Namespace != namespace {
		return "// " + u.Name + " external; skip"
	}
	s := "type " + GoName(u) + " struct {\n"
	s += "\t//union\n"
	for _, n := range u.Fields {
		s += FieldToGo(n) + "\n"
	}
	s += "}\n"
	return s
}

func TypeToGo(t *TypeInfo, isArg bool) string {
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
			s = "unsafe.Pointer"
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
			s += TypeToGo(t.ParamTypes[0], false)
		case GPtrArray:
			s += "[]*"
			s += TypeToGo(t.ParamTypes[0], false)
		case GByteArray:
			s += "[]byte"
		default:
			panic(fmt.Errorf("unknown array type %d", t.ArrayType))
		}
	case TagInterface:
		if t.Interface.Namespace != namespace {
			s += strings.ToLower(t.Interface.Namespace) + "."
		}
		if isArg {		// arguments become the equivalent interfaces
			if t.Type == TypeObject {
				s += "I"
			}
			// TODO structs and unions?
		}
		s += t.Interface.Name
	case TagGList:
		s += "[]"
		if t.ParamTypes[0].GContainerStorePointer() {
			s += "*"
		}
		s += TypeToGo(t.ParamTypes[0], false)
	case TagGSList:
		s += "[]"
		if t.ParamTypes[0].GContainerStorePointer() {
			s += "*"
		}
		s += TypeToGo(t.ParamTypes[0], false)
	case TagGHashTable:
		s += "map["
		if t.ParamTypes[0].GContainerStorePointer() {
			s += "*"
		}
		s += TypeToGo(t.ParamTypes[0], false)
		s += "]"
		if t.ParamTypes[1].GContainerStorePointer() {
			s += "*"
		}
		s += TypeToGo(t.ParamTypes[1], false)
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
func (t *TypeInfo) GContainerStorePointer() bool {
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
