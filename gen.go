// 23 june 2014
package main

import (
	"fmt"
	"os"
	"bytes"
)

func generate(ns Namespace) {
	b := new(bytes.Buffer)

	fmt.Fprintf(b, "package %s\n\nimport \"unsafe\"\nimport \"errors\"\n\n// ADD IMPORTS AND CGO DIRECTIVES HERE\n// BE SURE TO INCLUDE stdio.h\n\n", nsGoName(ns.Name))

	// enumerations
	// to avoid unnecessary typing, let's collect all value names
	// if, for any enum, at least one name is ambiguous, we require the first word of the enum name as a prefix
	namecount := map[string]int{}
	for _, n := range ns.TopLevelEnums {
		e := ns.Enums[n]
		if e.Namespace != ns.Name {		// skip foreign imports
			continue
		}
		for _, i := range e.Values {
			v := ns.Values[i]
			namecount[ns.GoName(v)]++
		}
	}
	for _, n := range ns.TopLevelEnums {
		e := ns.Enums[n]
		if e.Namespace != ns.Name {		// skip foreign imports
			continue
		}
		goName := ns.GoName(e)
		fmt.Fprintf(b, "type %s %s\n", goName, e.StorageType.BasicString())
		fmt.Fprintf(b, "const (\n")
		fgw := ""
		for _, i := range e.Values {
			v := ns.Values[i]
			if namecount[ns.GoName(v)] > 1 {
				fgw = firstGoWord(goName)
				break
			}
		}
		for _, i := range e.Values {
			v := ns.Values[i]
			fmt.Fprintf(b, "\t%s%s %s = C.%s\n",
				fgw, ns.GoName(v), goName, ns.CName(v))
		}
		fmt.Fprintf(b, ")\n")
		fmt.Fprintf(b, "\n")
	}

	// interfaces
	// we don't need to worry about implementations of methods for each object until we get to the objects themselves
	// we also don't need to worry about signals
	// we DO need to worry about prerequisite types, putting an I before object prerequisites
	for _, n := range ns.TopLevelInterfaces {
		ii := ns.Interfaces[n]
		if ii.Namespace != ns.Name {		// skip foreign imports
			continue
		}
		goName := ns.GoName(ii)
		fmt.Fprintf(b, "type %s interface {\n", goName)
		for _, p := range ii.Prerequisites {
			fmt.Fprintf(b, "\t%s\n", ns.GoIName(p))
		}
		for _, m := range ii.VFuncs {
			v := ns.VFuncs[m]
			fmt.Fprintf(b, "\tfunc %s\n", ns.GoFuncSig(v.CallableInfo))
		}
		fmt.Fprintf(b, "}\n")
		// TODO constants
		fmt.Fprintf(b, "\n")
	}

	// objects
	// all objects are either derived (embed the base class) or not (have a native member)
	// each object also gets the methods of the interfaces it implements
	// each object ALSO gets its own interface, to play into the whole polymorphism thing
	for _, n := range ns.TopLevelObjects {
		o := ns.Objects[n]
		if o.Namespace != ns.Name {		// skip foreign imports
			continue
		}
		goName := ns.GoName(o)
		goIName := ns.GoIName(o)
		fmt.Fprintf(b, "type %s struct {\n", goName)
		if o.Parent == -1 {		// base
			fmt.Fprintf(b, "\tnative unsafe.Pointer\n")
			fmt.Fprintf(b, "}\n")
			fmt.Fprintf(b, "func (c *%s) Native() uintptr {\n", goName)
			fmt.Fprintf(b, "\treturn uintptr(c.native)\n");
		} else {
			oo := ns.Objects[o.Parent]
			fmt.Fprintf(b, "\t%s\n", ns.GoName(oo))
		}
		fmt.Fprintf(b, "}\n")
		for _, m := range o.Methods {
			mm := ns.Functions[m]
			fmt.Fprintf(b, "%s\n", ns.wrap(mm, o, false, InterfaceInfo{}))
		}
		for _, ii := range o.Interfaces {
			iii := ns.Interfaces[ii]
			for _, m := range iii.Methods {
				mm := ns.Functions[m]
				fmt.Fprintf(b, "%s\n", ns.wrap(mm, o, true, iii))
			}
		}
		// TODO other methods
		fmt.Fprintf(b, "type %s interface {\n", goIName)
		if o.Parent != -1 {
			oo := ns.Objects[o.Parent]
			fmt.Fprintf(b, "\t%s\n", ns.GoIName(oo))
		}
		for _, ii := range o.Interfaces {
			iii := ns.Interfaces[ii]
			fmt.Fprintf(b, "\t%s\n", ns.GoName(iii))
		}
		for _, m := range o.Methods {
			f := ns.Functions[m]
			if f.IsMethod {			// only actual methods
				fmt.Fprintf(b, "\tfunc %s\n", ns.GoFuncSig(f.CallableInfo))
			}
		}
		fmt.Fprintf(b, "}\n")
		// TODO constants
		fmt.Fprintf(b, "\n")
	}

	os.Stdout.Write(b.Bytes())
}

// the rest of this file generates a wrapper function, taking care of converting special GLib constructs

var basicNames = map[TypeTag]string{
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

func (ns Namespace) argPrefix(arg ArgInfo, t TypeInfo) string {
	// no prefix needed
	if n, ok := basicNames[t.Tag]; ok {
		return fmt.Sprintf("\treal_%s := %s(%s)\n", arg.Name, n, arg.Name)
	}
	switch t.Tag {
	case TagBoolean:
		s := fmt.Sprintf("\treal_%s := C.gboolean(C.TRUE)\n", arg.Name)
		s += fmt.Sprintf("\tif !(%s) { real_%s = C.gboolean(C.FALSE)\n", arg.Name, arg.Name)
		return s
	case TagUTF8String, TagFilename:
		s := fmt.Sprintf("\treal_%s := (*C.gchar)(unsafe.Pointer(C.CString(%s)))\n", arg.Name, arg.Name)
		s += fmt.Sprintf("\tdefer C.free(unsafe.Pointer(real_%s))\n", arg.Name)
		return s
	case TagArray:
		// TODO
	case TagInterface:
		ctype := ns.CName(t.Interface)
		if t.Interface.Type == TypeEnum {
			return fmt.Sprintf("\treal_%s = (C.%s)(%s)\n", arg.Name, ctype, arg.Name)
		}
		return fmt.Sprintf("\treal_%s = (*C.%s)(unsafe.Pointer(%s.Native()))\n", arg.Name, ctype, arg.Name)
	case TagGList:
		s := fmt.Sprintf("\tvar real_%s *C.GList = nil\n", arg.Name)
		s += fmt.Sprintf("\tfor _, real_%s_val := range %s {\n", arg.Name, arg.Name)
		if ns.Types[t.ParamTypes[0]].GContainerStorePointer() {
			s += fmt.Sprintf("\t\treal_%s = C.g_list_prepend(real_%s, C.gpointer(unsafe.Pointer(real_%s_arg))\n", arg.Name, arg.Name, arg.Name)
		} else {
			// TODO floats fail this
			s += fmt.Sprintf("\t\treal_%s = C.g_list_prepend(real_%s, C.gpointer(unsafe.Pointer(uintptr(real_%s_arg)))\n", arg.Name, arg.Name, arg.Name)
		}
		s += "\t}\n"
		s += fmt.Sprintf("\treal_%s = C.g_list_reverse(real_%s)\n", arg.Name, arg.Name)
		// TODO bad for dynamic stuff
		s += fmt.Sprintf("\tdefer C.g_list_free(real_%s)\n", arg.Name)
		return s
	case TagGSList:
		// TODO
	case TagGHashTable:
		// TODO
	case TagGError:
		return fmt.Sprintf("\tvar real_%s *C.GError = nil\n", arg.Name)
	default:
		panic(fmt.Errorf("unknown tag type %d in argPrefix()", t.Tag))
	}
	return "\t//TODO\n"
}

func (ns Namespace) argSuffix(arg ArgInfo, t TypeInfo) string {
	if t.Tag == TagGError {
		s := fmt.Sprintf("\tif real_%s != nil {\n", arg.Name)
		s += fmt.Sprintf("\t\tcmsg_%s := (*C.char)(unsafe.Pointer(real_%s.message))\n", arg.Name)
		s += fmt.Sprintf("\t\t%s = errors.New(C.GoString(cmsg_%s)\n", arg.Name, arg.Name)
		s += fmt.Sprintf("\t}\n")
		return s
	}
	return ""			// no extra cleanup needed
}

func (ns Namespace) retconv(expr string, t TypeInfo) string {
	switch t.Tag {
	case TagVoid:
		if t.IsPointer {
			// TODO
		}
		return expr		// no return
	case TagBoolean:
		return "(" + expr + ") != C.FALSE"
	case TagUTF8String,TagFilename:
		return "C.GoString((*C.char)(unsafe.Pointer(" + expr + ")))"
	case TagArray:
		// TODO
	case TagInterface:
		if t.IsPointer {		// objects
			return fmt.Sprintf("&%s{}; ret.native = unsafe.Pointer(%s)", ns.GoName(t.Interface), expr)
		}
		// fall through to the bottom, which does what we want
	case TagGList:
		// TODO
	case TagGSList:
		// TODO
	case TagGHashTable:
		// TODO
	case TagGError:
		// TODO
	}
	// anything else? take a guess... (correct for basic types)
	return fmt.Sprintf("(%s)(%s)", ns.TypeValueToGo(t, false), expr)
}

func (ns Namespace) wrap(method FunctionInfo, to ObjectInfo, isInterface bool, iface InterfaceInfo) string {
	s := "func "
	prefix := ""
	suffix := ""
	// method receivers aren't listed in the arguments; we have to fake it
	if method.IsMethod {
		// make a fake receiver
		receiver := ArgInfo{
			BaseInfo:		BaseInfo{
				Namespace:	ns.Name,
				Name:		"this",		// let's hope nothing uses this name
			},
		}
		rtype := TypeInfo{
			BaseInfo:		BaseInfo{
				Namespace:	ns.Name,
			},
			IsPointer:		true,
			Tag:			TagInterface,
			Interface:		to.BaseInfo,
		}
		itype := rtype
		if isInterface {
			itype.Interface = iface.BaseInfo
		}
		s += "("
		prefix += ns.argPrefix(receiver, itype)
		suffix = ns.argSuffix(receiver, itype) + suffix
		s += ns.ArgValueToGo(receiver, rtype, false)
		s += ") "
	}
	// disambiguate between constructors
	// a more Go-like way would be to insert the type name after the New but before anything else :/ conformal/gotk3 does it this way so meh
	if (method.Flags & FunctionIsConstructor) != 0 {
		s += ns.GoName(to)
	}
	s += ns.GoName(method) + "("
	for i := 0; i < len(method.Args); i++ {
		arg := ns.Args[method.Args[i]]
		prefix += ns.argPrefix(arg, ns.Types[arg.Type])
		suffix = ns.argSuffix(arg, ns.Types[arg.Type]) + suffix
		s += ns.ArgToGo(method.Args[i])
		s += ", "
	}
	s += ") "
	ret := ns.TypeToGo(method.ReturnType)
	if ret != "" {
		s += "(ret " + ret + ") "
	}
	s += "{\n"
	s += prefix
	s += "\t"
	j := "C." + ns.CName(method) + "("
	if method.IsMethod {
		j += "real_this, "
	}
	for i := 0; i < len(method.Args); i++ {
		arg := ns.Args[method.Args[i]]
		j += "real_" + arg.Name + ", "
	}
	j += ")"
	if ret != "" {
		s += "ret = "
	}
	s += ns.retconv(j, ns.Types[method.ReturnType])
	s += "\n"
	s += suffix
	if ret != "" {
		s += "\treturn ret\n"
	}
	s += "}"
	return s
}
