// 23 june 2014
package main

import (
	"fmt"
	"os"
	"bytes"
)

func generate(ns Namespace) {
	b := new(bytes.Buffer)

	fmt.Fprintf(b, "package %s\n\nimport \"unsafe\"\nimport \"errors\"\nimport \"math\"\n\n// ADD IMPORTS AND CGO DIRECTIVES HERE\n// BE SURE TO INCLUDE stdio.h\n\n", nsGoName(ns.Name))

	// enumerations
	// to avoid unnecessary typing, let's collect all value names
	// if, for any enum, at least one name is ambiguous, we require the first word of the enum name as a prefix
	namecount := map[string]int{}
	for _, e := range ns.TopLevelEnums {
		if e.Namespace != namespace {		// skip foreign imports
			continue
		}
		for _, v := range e.Values {
			namecount[GoName(v)]++
		}
	}
	for _, e := range ns.TopLevelEnums {
		if e.Namespace != namespace {		// skip foreign imports
			continue
		}
		goName := GoName(e)
		fmt.Fprintf(b, "type %s %s\n", goName, e.StorageType.BasicString())
		fmt.Fprintf(b, "const (\n")
		fgw := ""
		for _, v := range e.Values {
			if namecount[GoName(v)] > 1 {
				fgw = firstGoWord(goName)
				break
			}
		}
		for _, v := range e.Values {
			fmt.Fprintf(b, "\t%s%s %s = C.%s\n",
				fgw, GoName(v), goName, CName(v))
		}
		fmt.Fprintf(b, ")\n")
		fmt.Fprintf(b, "\n")
	}

	// interfaces
	// we don't need to worry about implementations of methods for each object until we get to the objects themselves
	// we also don't need to worry about signals
	// we DO need to worry about prerequisite types, putting an I before object prerequisites
	for _, ii := range ns.TopLevelInterfaces {
		if ii.Namespace != namespace {		// skip foreign imports
			continue
		}
		goName := GoName(ii)
		fmt.Fprintf(b, "type %s interface {\n", goName)
		for _, p := range ii.Prerequisites {
			fmt.Fprintf(b, "\t%s\n", GoIName(p))
		}
		for _, v := range ii.VFuncs {
			fmt.Fprintf(b, "\tfunc %s\n", GoFuncSig(v.CallableInfo))
		}
		fmt.Fprintf(b, "}\n")
		// TODO constants
		fmt.Fprintf(b, "\n")
	}

	// objects
	// all objects are either derived (embed the base class) or not (have a native member)
	// each object also gets the methods of the interfaces it implements
	// each object ALSO gets its own interface, to play into the whole polymorphism thing
	for _, o := range ns.TopLevelObjects {
		if o.Namespace != namespace {		// skip foreign imports
			continue
		}
		goName := GoName(o)
		goIName := GoIName(o)
		fmt.Fprintf(b, "type %s struct {\n", goName)
		if o.Parent == nil {		// base
			fmt.Fprintf(b, "\tnative unsafe.Pointer\n")
			fmt.Fprintf(b, "}\n")
			fmt.Fprintf(b, "func (c *%s) Native() uintptr {\n", goName)
			fmt.Fprintf(b, "\treturn uintptr(c.native)\n");
		} else {
			fmt.Fprintf(b, "\t%s\n", GoName(o.Parent))
		}
		fmt.Fprintf(b, "}\n")
		for _, mm := range o.Methods {
			fmt.Fprintf(b, "%s\n", ns.wrap(mm, o.BaseInfo, false, nil))
		}
		for _, iii := range o.Interfaces {
			for _, mm := range iii.Methods {
				fmt.Fprintf(b, "%s\n", ns.wrap(mm, o.BaseInfo, true, iii))
			}
		}
		// TODO other methods
		fmt.Fprintf(b, "type %s interface {\n", goIName)
		if o.Parent != nil {
			fmt.Fprintf(b, "\t%s\n", GoIName(o.Parent))
		}
		for _, iii := range o.Interfaces {
			fmt.Fprintf(b, "\t%s\n", GoName(iii))
		}
		for _, f := range o.Methods {
			if f.IsMethod {			// only actual methods
				fmt.Fprintf(b, "\tfunc %s\n", GoFuncSig(f.CallableInfo))
			}
		}
		fmt.Fprintf(b, "}\n")
		// TODO constants
		fmt.Fprintf(b, "\n")
	}

	// structures
	for _, s := range ns.TopLevelStructs {
		if s.Namespace != namespace {		// skip foreign imports
			continue
		}
		if s.IsClassStruct {				// skip GObject boilerplate
			continue
		}
		if s.Foreign {		// TODO debugging
			fmt.Fprintf(b, "// foreign\n")
		}
		goName := GoName(s)
		if len(s.Fields) == 0 && bytes.HasSuffix([]byte(goName), []byte("Private")) {
			// skip opaque private structures (implementation details that are slowly being eliminated)
			// this should be safe; very few nonempty privates are left that it doesn't matter (and let's bind glib.Private anyway, just to be safe)
			continue
		}
		fmt.Fprintf(b, "type %s struct {\n", goName)
		for _, f := range s.Fields {
			// TODO substitute TypeToGo()
			fmt.Fprintf(b, "\t%s %s\n", GoName(f), TypeToGo(f.Type, false))
		}
		fmt.Fprintf(b, "}\n")
		// TODO conversion functions
		for _, mm := range s.Methods {
			fmt.Fprintf(b, "%s\n", ns.wrap(mm, s.BaseInfo, false, nil))
		}
		fmt.Fprintf(b, "\n")
	}

	os.Stdout.Write(b.Bytes())
}

func (ns Namespace) wrap(method *FunctionInfo, to BaseInfo, isInterface bool, iface *InterfaceInfo) string {
	s := "func "
	prefix := ""
	suffix := ""
	arglist := ""
	// method receivers aren't listed in the arguments; we have to fake it
	if method.IsMethod {
		bi := BaseInfo{}
		if isInterface {
			bi = iface.BaseInfo
		}
		receiver := receiverArg(to, isInterface, bi)
		s += "("
		prefix += receiver.Prefix()
		suffix = receiver.Suffix() + suffix
		arglist += receiver.GoArg() + ", "
		s += receiver.GoDecl()
		s += ") "
	}
	// disambiguate between constructors
	// a more Go-like way would be to insert the type name after the New but before anything else :/ conformal/gotk3 does it this way so meh
	if (method.Flags & FunctionIsConstructor) != 0 {
		s += GoName(to)
	}
	s += GoName(method) + "("
	for i := 0; i < len(method.Args); i++ {
		arg := argumentArg(method.Args[i])
		prefix += arg.Prefix()
		suffix = arg.Suffix() + suffix
		arglist += arg.GoArg() + ", "
		s += arg.GoDecl()
		s += ", "
	}
	s += ") "
	retarg := returnArg(method.ReturnType)
	prefix += retarg.Prefix()
	suffix = retarg.Suffix() + suffix
	s += retarg.GoDecl()
	if len(retarg.GoDecl()) != 0 {
		s += " "
	}
	s += "{\n"
	s += prefix
	s += "\t" + retarg.GoCall("C." + CName(method) + "(" + arglist + ")") + "\n"
	s += suffix
	s += retarg.GoRet()
	s += "}"
	return s
}
