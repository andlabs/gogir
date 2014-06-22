// 21 june 2014
package main

import (
	"fmt"
	"os"
	"unsafe"
)

// #cgo pkg-config: gobject-introspection-1.0
// #include <girepository.h>
import "C"

var typenames = map[C.GIInfoType]string{
	C.GI_INFO_TYPE_INVALID:		"invalid",
	C.GI_INFO_TYPE_FUNCTION:		"function",
	C.GI_INFO_TYPE_CALLBACK:		"callback",
	C.GI_INFO_TYPE_STRUCT:		"struct",
	C.GI_INFO_TYPE_BOXED:			"boxed",
	C.GI_INFO_TYPE_ENUM:			"enum",
	C.GI_INFO_TYPE_FLAGS:			"flags",
	C.GI_INFO_TYPE_OBJECT:		"object",
	C.GI_INFO_TYPE_INTERFACE:		"interface",
	C.GI_INFO_TYPE_CONSTANT:		"constant",
	C.GI_INFO_TYPE_INVALID_0:		"invalid0",
	C.GI_INFO_TYPE_UNION:			"union",
	C.GI_INFO_TYPE_VALUE:			"value",
	C.GI_INFO_TYPE_SIGNAL:		"signal",
	C.GI_INFO_TYPE_VFUNC:			"vfunc",
	C.GI_INFO_TYPE_PROPERTY:		"property",
	C.GI_INFO_TYPE_FIELD:			"field",
	C.GI_INFO_TYPE_ARG:			"arg",
	C.GI_INFO_TYPE_TYPE:			"type",
	C.GI_INFO_TYPE_UNRESOLVED:	"unresolved",
}

var (
	_gtkns = [...]C.gchar{'G', 't', 'k', 0}
	gtkns = &_gtkns[0]
)

func fromgstr(str *C.gchar) string {
	return C.GoString((*C.char)(unsafe.Pointer(str)))
}

func main() {
	var err *C.GError = nil

	gtk := C.g_irepository_require(nil, gtkns, nil, 0, &err)
	if gtk == nil {
		fmt.Fprintf(os.Stderr, "error opening GTK+ gi repository: %s", err.message)
		os.Exit(1)
	}
	ninfo := C.g_irepository_get_n_infos(nil, gtkns)
	for i := C.gint(0); i < ninfo; i++ {
		var iter C.GIAttributeIter		// sets first member (really all members) to zero
		var name, value *C.char

		info := C.g_irepository_get_info(nil, gtkns, i)
		fmt.Printf("%s %s attribs { \n",
			typenames[C.g_base_info_get_type(info)],
			fromgstr(C.g_base_info_get_name(info)))
		for C.g_base_info_iterate_attributes(info, &iter, &name, &value) != C.FALSE {
			fmt.Printf("\t%q = %q,\n", C.GoString(name), C.GoString(value))
		}
		fmt.Printf("}\n")
		C.g_base_info_unref(info);
	}
}
