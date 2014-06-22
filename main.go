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
		info := C.g_irepository_get_info(nil, gtkns, i)
		fmt.Printf("type:%d name:%s\n",
			C.g_base_info_get_type(info),
			fromgstr(C.g_base_info_get_name(info)))
		C.g_base_info_unref(info);
	}
}
