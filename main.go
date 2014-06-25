// 23 june 2014
package main

import (
	"fmt"
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
	ret := ci.ReturnType.GoType(false)
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
	s := "const " + GoName(c) + " " + c.Type.GoType(false) + " = "
	s += "C." + CName(c)
	return s
}

func FieldToGo(f *FieldInfo) string {
	if f.Namespace != namespace {
		return "\t// " + f.Name + " external; skip"
	}
	s := "\t" + GoName(f) + " " + f.Type.GoType(false)
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

// for GList, GSList, and GHashTable, whether the stored type is a pointer is not stored; use this function to find out
// interfaces become Go interfaces which are /references/, so don't make htem pointers either
func (t *TypeInfo) GContainerStorePointer() bool {
	return t.Tag == TagInterface && t.Interface.Type != TypeInterface //TODO && t.Interface.Type != TypeEnum
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
