// 23 june 2014
package main

import (
	"fmt"
	"os"
	"bytes"
)

func generate(ns Namespace) {
	b := new(bytes.Buffer)

	fmt.Fprintf(b, "package %s\n\n// ADD IMPORTS AND CGO DIRECTIVES HERE\n\n", nsGoName(ns.Name))

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
		fmt.Fprintf(b, "\n")
	}

	os.Stdout.Write(b.Bytes())
}

