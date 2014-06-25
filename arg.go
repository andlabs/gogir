// 24 june 2014
package main

import (
	"fmt"
	"strings"
)

// An Arg can be one fo three things:
// - a receiver
// - an actual argument
// - a return value
// Each of these have slightly different semantics. This file clears up all the bases.

// TODO put this in gen.go and use it
var namespace string

type Arg struct {
	Name		string
	Type			*TypeInfo
	Polymorphic	bool
	RealType		*TypeInfo
	Receiver		bool
	Arg			Direction
	Transfer		Transfer
	Return		bool
}

// receivers are (this Type) for enums and (this *Type) for everything else
// receivers can also be polymorphic (interface functions), which affects their generated prefix/suffix
func receiverArg(to BaseInfo, polymorphic bool, real BaseInfo) Arg {
	a := Arg{
		Name:		"this",
		Type:		&TypeInfo{
			BaseInfo:		BaseInfo{
				Namespace:	namespace,
			},
			IsPointer:		to.Type != TypeEnum,
			Tag:			TagInterface,
			Interface:		to,
		},
		Polymorphic:	polymorphic,
		Receiver:		true,
	}
	if a.Polymorphic {
		a.RealType = new(TypeInfo)
		*a.RealType = *a.Type
		a.RealType.Interface = real
	}
	return a
}

func argumentArg(arg *ArgInfo) Arg {
	return Arg{
		Name:	arg.Name,
		Type:	arg.Type,
		Arg:		arg.Direction,
		Transfer:	arg.OwnershipTransfer,
	}
}

func returnArg(t *TypeInfo) Arg {
	return Arg{
		Name:	"ret",
		Type:	t,
		Return:	true,
	}
}

var basicCNames = map[TypeTag]string{
	TagVoid:			"unsafe.Pointer",
	TagInt8:			"C.gint8",
	TagUint8:			"C.guint8",
	TagInt16:			"C.gint16",
	TagUint16:		"C.guint16",
	TagInt32:			"C.gint32",
	TagUint32:		"C.guint32",
	TagInt64:			"C.gint64",
	TagUint64:		"C.guint64",
	TagFloat:			"C.gfloat",
	TagDouble:		"C.gdouble",
	TagGType:		"C.GType",
	TagUnichar:		"C.gunichar",
}

var basicGoNames = map[TypeTag]string{
	// no void; needs special handling
	TagInt8:			"int8",
	TagUint8:			"uint8",
	TagInt16:			"int16",
	TagUint16:		"uint16",
	TagInt32:			"int32",
	TagUint32:		"uint32",
	TagInt64:			"int64",
	TagUint64:		"uint64",
	TagFloat:			"float32",
	TagDouble:		"float64",
	// no GType; needs special handling
	TagUnichar:		"rune",
}

func (t *TypeInfo) CType() string {
	if t.Tag == TagVoid && !t.IsPointer {
		return ""
	}
	if s, ok := basicCNames[t.Tag]; ok {
		return s
	}
	switch t.Tag {
	case TagBoolean:	// not in basicCNames because requires special handling
		return "C.gboolean"
	case TagUTF8String, TagFilename:
		return "*C.gchar"
	case TagArray:
		switch t.ArrayType {
		case CArray:
			// TODO change [] to *
			return "[]" + t.ParamTypes[0].CType()
		case GArray:
			return "*C.GArray"
		case GPtrArray:
			return "*C.GPtrArray"
		case GByteArray:
			return "*C.GByteArray"
		default:
			panic(fmt.Errorf("unknown array type %d in TypeInfo.CType()", t.ArrayType))
		}
	case TagInterface:
		s := "C." + t.Interface.Namespace + t.Interface.Name
		if t.IsPointer {
			s = "*" + s
		}
		return s
	case TagGList:
		return "*C.GList"
	case TagGSList:
		return "*C.GSList"
	case TagGHashTable:
		return "*C.GHashTable"
	case TagGError:
		return "*C.GError"
	}
	panic(fmt.Errorf("unknown tag type %d in TypeInfo.CType()", t.Tag))
}

func (t *TypeInfo) GoType(arg bool) string {
	prefix := ""
	if t.IsPointer {
		prefix = "*"
	}
	if t.Tag == TagVoid && !t.IsPointer {
		return ""
	}
	if s, ok := basicGoNames[t.Tag]; ok {
		return prefix + s
	}
	switch t.Tag {
	case TagVoid:		// not in basicGoNames because requires special handling
		// TODO should this be interface[}?
		return "unsafe.Pointer"		// !t.isPointer case handled above
	case TagBoolean:
		return prefix + "bool"
	case TagGType:
		if namespace != "gobject" {
			return prefix + "gobject.GType"
		}
		return prefix + "GType"
	case TagUTF8String, TagFilename:
		// ignore pointer
		return "string"
	case TagArray:
		// ignore pointer if not GPtrArray; force []byte for GByteArray
		// arg should not be carried below the first recursive call
		switch t.ArrayType {
		case CArray, GArray:
			return "[]" + t.ParamTypes[0].GoType(false)
		case GPtrArray:
			return "[]*" + t.ParamTypes[0].GoType(false)
		case GByteArray:
			return "[]byte"
		}
		panic(fmt.Errorf("unknown array type %d in TypeInfo.GoType()", t.ArrayType))
	case TagInterface:
		s := t.Interface.Name
		isInterface := t.Interface.Type == TypeInterface
		if arg && t.Interface.Type == TypeObject {	// arguments are the mirroring interface type
			s = "I" + s
			isInterface = true
		}
		if isInterface {		// wipe pointer
			prefix = ""
		}
		if t.Interface.Namespace != namespace {
			s = strings.ToLower(t.Interface.Namespace) + "." + s
		}
		s = prefix + s
		return s
	case TagGList, TagGSList:
		// override prefix so that * is only added if a non-interface, non-enum object is being stored
		t0 := t.ParamTypes[0]
		prefix = ""
		if t0.GContainerStorePointer() {
			prefix = "*"
		}
		// arg should not be carried below the first recursive call
		return "[]" + prefix + t.ParamTypes[0].GoType(false)
	case TagGHashTable:
		// see above on overriding pointers
		t0 := t.ParamTypes[0]
		prefixa := ""
		if t0.GContainerStorePointer() {
			prefixa = "*"
		}
		t1 := t.ParamTypes[0]
		prefixb := ""
		if t1.GContainerStorePointer() {
			prefixb = "*"
		}
		// arg should not be carried below the first recursive call
		return "map[" + prefixa + t0.GoType(false) + "]" + prefixb + t1.GoType(false)
	case TagGError:
		// ignore pointer
		return "error"
	}
	panic(fmt.Errorf("unknown tag type %d in TypeInfo.CType()", t.Tag))
}

func (a Arg) listIn(ss string) string {
	s := fmt.Sprintf("\tvar real_%s *C.G%sList = nil\n", a.Name, strings.ToUpper(ss))
	realval := "real_" + a.Name + "_val"
	s += fmt.Sprintf("\tfor _, %s := range %s {\n", realval, a.Name)
	format := "\t\treal_%s = C.g_%slist_prepend(real_%s, C.gpointer(unsafe.Pointer(%s))\n"
	inner := "uintptr(" + realval + ")"
	ptype := a.Type.ParamTypes[0]
	if ptype.Tag == TagInterface {
		switch ptype.Interface.Type {
		case TypeInterface, TypeObject:
			inner = realval + ".Native()"
		case TypeStruct:
			s += "\t\txdummy := " + realval + "._cstruct()\n"
			s += "\t\tdefer C.free(xdummy)\n"
			inner = "xdummy"
		case TypeUnion:
			// TODO
		}
		// enum just keeps the default
	} else if ptype.Tag == TagFloat {
		inner = "uintptr(math.Float32bits(" + realval + "))"
	} else if ptype.Tag == TagDouble {
		inner = "uintptr(math.Float64bits(" + realval + "))"
	}
	s += fmt.Sprintf(format, a.Name, ss, a.Name, inner)
	s += "\t}\n"
	s += fmt.Sprintf("\treal_%s = C.g_%slist_reverse(real_%s)\n", ss, a.Name, a.Name)
	s += fmt.Sprintf("\tdefer C.g_%slist_free(real_%s)\n", ss, a.Name)
	return s
}

func (a Arg) Prefix() string {
	if a.Receiver {
		// should always be an object type
		format := "\treal_%s := (%%s)(%s.native)\n"
		format = fmt.Sprintf(format, a.Name, a.Name)
		if a.Polymorphic {
			return fmt.Sprintf(format, a.RealType.CType())
		}
		return fmt.Sprintf(format, a.Type.CType())
	}

	t := a.Type

	if a.Arg == Out {
		return fmt.Sprintf("\tvar real_%s %s\n", a.Name, t.CType())
	}
	if a.Return {
		if a.Type.Tag == TagVoid && !a.Type.IsPointer {
			return ""
		}
		return fmt.Sprintf("\tvar real_%s %s\n", a.Name, t.CType())
	}

	if s, ok := basicCNames[t.Tag]; ok {
		return fmt.Sprintf("\treal_%s := %s(%s)\n", a.Name, s, a.Name)
	}

	switch t.Tag {
	case TagBoolean:
		s := fmt.Sprintf("\treal_%s := C.gboolean(C.TRUE)\n", a.Name)
		s += fmt.Sprintf("\tif !(%s) { real_%s = C.gboolean(C.FALSE) }\n", a.Name, a.Name)
		return s
	case TagUTF8String, TagFilename:
		s := fmt.Sprintf("\treal_%s := (*C.gchar)(unsafe.Pointer(C.CString(%s)))\n", a.Name, a.Name)
		s += fmt.Sprintf("\tdefer C.free(unsafe.Pointer(real_%s))\n", a.Name)
		return s
	case TagArray:
		return "// TODO"
	case TagInterface:
		ctype := t.CType()
		format := "\treal_%s = (*C.%s)(unsafe.Pointer(%s.Native()))\n"
		if t.Interface.Type == TypeEnum {		// enums are by value
			format = "\treal_%s = (C.%s)(%s)\n"
		}
		return fmt.Sprintf(format, a.Name, ctype, a.Name)
	case TagGList:
		return a.listIn("")
	case TagGSList:
		return a.listIn("s")
	case TagGHashTable:
		return "// TODO"
	case TagGError:
		return "// TODO"
//		return fmt.Sprintf("\tvar real_%s *C.GError = nil\n", arg.Name)
	}
	panic(fmt.Errorf("unknown tag type %d in Arg.Prefix()", t.Tag))
}

func (a Arg) Suffix() string {
	if (!a.Receiver && !a.Return && a.Arg == In) || a.Receiver {
		// nothing to do here
		return ""
	}

	t := a.Type
	realname := a.Name
	if !a.Return {
		realname = "*" + realname
	}

	if s, ok := basicGoNames[t.Tag]; ok {
		return fmt.Sprintf("\t%s = (%s)(real_%s)\n", realname, s, a.Name)
	}

	switch t.Tag {
	case TagVoid:
		if t.IsPointer {
			return fmt.Sprintf("\t%s = unsafe.Pointer(real_%s)\n", realname, a.Arg)
		}
		return ""
	case TagBoolean:
		return fmt.Sprintf("\t%s = real_%s != C.gboolean(C.FALSE)\n", realname, a.Name)
	case TagGType:
		return "// TODO"
	case TagUTF8String, TagFilename:
		return fmt.Sprintf("\t%s = C.GoString((*C.char)(unsafe.Pointer(%s)))\n", realname, a.Name)
	case TagArray:
		return "// TODO"
	case TagInterface:
		s := t.GoType(false)
		if t.IsPointer {		// objects
			s = s[1:]		// strip *
			return fmt.Sprintf("\t%s = &%s{}; %s.native = unsafe.Pointer(real_%s)\n", realname, s, realname, a.Name)
		}
		return fmt.Sprintf("\t%s = (%s)(real_%s)\n", realname, s, a.Name)
	case TagGList:
return"TODO"//		return a.listOut("")
	case TagGSList:
return"TODO"//		return a.listOut("s")
	case TagGHashTable:
		return "// TODO"
	case TagGError:
		s := fmt.Sprintf("\t%s = nil\n", realname)
		s += fmt.Sprintf("\tif real_%s != nil {\n", a.Name)
		msg := fmt.Sprintf("(*C.char)(unsafe.Pointer(real_%s.message))", a.Name)
		s += fmt.Sprintf("\t\t%s = errors.New(%s)\n", realname, msg)
		s += "\t}\n"
		return s
	}
	panic(fmt.Errorf("unknown tag type %d in Arg.Suffix()", t.Tag))
}

func (a Arg) GoDecl() string {
	if a.Return {
		if a.Type.Tag == TagVoid && !a.Type.IsPointer {
			return ""
		}
		return "(" + a.Name + " " + a.Type.GoType(false) + ")"
	}
	return a.Name + " " + a.Type.GoType(false)
}

func (a Arg) GoArg() string {
	s := "real_" + a.Name
	if a.Arg == Out || a.Arg == InOut {
		return "&" + s
	}
	return s
}

func (a Arg) GoCall(expr string) string {
	if !a.Return {
		return ""
	}
	if a.Type.Tag == TagVoid && !a.Type.IsPointer {
		return expr
	}
	return a.GoArg() + " = " + expr
}

func (a Arg) GoRet() string {
	if a.Type.Tag == TagVoid && !a.Type.IsPointer {
		return ""
	}
	return "\treturn " + a.Name + "\n"
}
