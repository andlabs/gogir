// 23 june 2014
package main

import (
	"strings"
	"unicode"
)

// this file deals with the C and Go names that something should take

func nsCFuncName(ns string) string {
	if ns == "GObject" || ns == "GLib" || ns == "GModule" || ns == "Gio" || ns == "GIRepository" {		// manual overrides
		return "g_"
	}
	out := ""
	for _, r := range ns {
		if unicode.IsUpper(r) {
			out += "_"
			out += string(unicode.ToLower(r))
		}
	}
	if out[0] == '_' {		// strip leading _
		out = out[1:]
	}
	return out + "_"
}

func nsCConstName(ns string) string {
	return strings.ToUpper(nsCFuncName(ns))
}

func (ns Namespace) CName(i Info) string {
	b := i.baseInfo()
	// first, see if there's a "c:identifier" attribute
	if ident, ok := b.Attributes["c:identifier"]; ok {
		return ident
	}
	// now do type-specific options
	switch x := i.(type) {
	case FunctionInfo:
		return x.Symbol
	case VFuncInfo:
		return nsCFuncName(ns.Name + x.Name)
	case CallableInfo:
		return nsCFuncName(ns.Name + x.Name)
	case ConstantInfo:
		return nsCConstName(ns.Name + x.Name)
	case ValueInfo:
		return nsCConstName(ns.Name + x.Name)
	}
	// fall back to a guess/the correct answer for objects, interfaces, structs, and unions
	return ns.Name + b.Name
}

// the Go package name is just the first letter lowercase
// exception: gobject, glib, gmodule, and girepository
func nsGoName(ns string) string {
	if ns == "GObject" || ns == "GLib" || ns == "GModule" || ns == "GIRepository" {
		return strings.ToLower(ns)
	}
	nns := []rune(ns)
	nns[0] = unicode.ToLower(nns[0])
	return string(nns)
}

// for names that wouldn't already be in canonical form, convert second and later characters to uppercase, removing underscoress
func nsGoFieldValueName(ns string) string {
	out := ""
	first := true
	for _, c := range ns {
		if first {		// force uppercase
			out += string(unicode.ToUpper(c))
			first = false
			continue
		}
		if c == '_' {		// make _ start a new word
			first = true
			continue
		}
		out += string(unicode.ToLower(c))
	}
	return out
}

func (ns Namespace) GoName(i Info) string {
	b := i.baseInfo()
	// first, see if the namespace is different
	nsprefix := ""
	if b.Namespace != ns.Name {
		nsprefix = nsGoName(b.Namespace) + "."
	}
	// now do type-specific options
	switch x := i.(type) {
	case EnumInfo:
		return nsprefix + x.Name
	case InterfaceInfo:
		return nsprefix + x.Name
	case ObjectInfo:
		return nsprefix + x.Name
	case StructInfo:
		return nsprefix + x.Name
	case UnionInfo:
		return nsprefix + x.Name
	}
	// fall back to a guess/the correct answer for values, fields, and what not
	return nsprefix + nsGoFieldValueName(b.Name)
}
