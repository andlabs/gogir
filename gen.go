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
			fmt.Fprintf(b, "%s\n", ns.wrap(mm, o.BaseInfo, false, InterfaceInfo{}))
		}
		for _, ii := range o.Interfaces {
			iii := ns.Interfaces[ii]
			for _, m := range iii.Methods {
				mm := ns.Functions[m]
				fmt.Fprintf(b, "%s\n", ns.wrap(mm, o.BaseInfo, true, iii))
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

	// structures
	for _, n := range ns.TopLevelStructs {
		s := ns.Structs[n]
		if s.Namespace != ns.Name {		// skip foreign imports
			continue
		}
		if s.IsClassStruct {				// skip GObject boilerplate
			continue
		}
		if s.Foreign {		// TODO debugging
			fmt.Fprintf(b, "// foreign\n")
		}
		goName := ns.GoName(s)
		if len(s.Fields) == 0 && bytes.HasSuffix([]byte(goName), []byte("Private")) {
			// skip opaque private structures (implementation details that are slowly being eliminated)
			// this should be safe; very few nonempty privates are left that it doesn't matter (and let's bind glib.Private anyway, just to be safe)
			continue
		}
		fmt.Fprintf(b, "type %s struct {\n", goName)
		for _, m := range s.Fields {
			f := ns.Fields[m]
			// TODO substitute TypeToGo()
			fmt.Fprintf(b, "\t%s %s\n", ns.GoName(f), ns.TypeToGo(f.Type))
		}
		fmt.Fprintf(b, "}\n")
		// TODO conversion functions
		for _, m := range s.Methods {
			mm := ns.Functions[m]
			fmt.Fprintf(b, "%s\n", ns.wrap(mm, s.BaseInfo, false, InterfaceInfo{}))
		}
		fmt.Fprintf(b, "\n")
	}

	os.Stdout.Write(b.Bytes())
}

func (ns Namespace) wrap(method FunctionInfo, to BaseInfo, isInterface bool, iface InterfaceInfo) string {
	namespace = ns.Name
	s := "func "
	prefix := ""
	suffix := ""
	arglist := ""
	// method receivers aren't listed in the arguments; we have to fake it
	if method.IsMethod {
		receiver := receiverArg(to, isInterface, iface.BaseInfo)
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
		s += ns.GoName(to)
	}
	s += ns.GoName(method) + "("
	for i := 0; i < len(method.Args); i++ {
		arg := argumentArg(ns.Args[method.Args[i]], ns)
		prefix += arg.Prefix()
		suffix = arg.Suffix() + suffix
		arglist += arg.GoArg() + ", "
		s += arg.GoDecl()
		s += ", "
	}
	s += ") "
	retarg := returnArg(ns.Types[method.ReturnType], ns)
	prefix += retarg.Prefix()
	suffix = retarg.Suffix() + suffix
	s += retarg.GoDecl()
	if len(retarg.GoDecl()) != 0 {
		s += " "
	}
	s += "{\n"
	s += prefix
	s += "\t" + retarg.GoCall("C." + ns.CName(method) + "(" + arglist + ")") + "\n"
	s += suffix
	s += retarg.GoRet()
	s += "}"
	return s
}
