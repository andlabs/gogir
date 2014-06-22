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

var typefuncs = map[C.GIInfoType]func(*C.GIBaseInfo){
	C.GI_INFO_TYPE_INVALID:		nil,
	C.GI_INFO_TYPE_FUNCTION:		function,
	C.GI_INFO_TYPE_CALLBACK:		callable,
	C.GI_INFO_TYPE_STRUCT:		nil,
	C.GI_INFO_TYPE_BOXED:			nil,
	C.GI_INFO_TYPE_ENUM:			nil,
	C.GI_INFO_TYPE_FLAGS:			nil,
	C.GI_INFO_TYPE_OBJECT:		nil,
	C.GI_INFO_TYPE_INTERFACE:		nil,
	C.GI_INFO_TYPE_CONSTANT:		nil,
	C.GI_INFO_TYPE_INVALID_0:		nil,
	C.GI_INFO_TYPE_UNION:			nil,
	C.GI_INFO_TYPE_VALUE:			nil,
	C.GI_INFO_TYPE_SIGNAL:		signal,
	C.GI_INFO_TYPE_VFUNC:			callable,
	C.GI_INFO_TYPE_PROPERTY:		nil,
	C.GI_INFO_TYPE_FIELD:			nil,
	C.GI_INFO_TYPE_ARG:			arg,
	C.GI_INFO_TYPE_TYPE:			nil,
	C.GI_INFO_TYPE_UNRESOLVED:	nil,
}

var (
	_gtkns = [...]C.gchar{'G', 't', 'k', 0}
	gtkns = &_gtkns[0]
)

func fromgstr(str *C.gchar) string {
	return C.GoString((*C.char)(unsafe.Pointer(str)))
}

func fromgbool(b C.gboolean) string {
	return fmt.Sprint(b != C.FALSE)
}

var directions = map[C.GIDirection]string{
	C.GI_DIRECTION_IN:		"in",
	C.GI_DIRECTION_OUT:		"out",
	C.GI_DIRECTION_INOUT:		"inout",
}

var transfers = map[C.GITransfer]string{
	C.GI_TRANSFER_NOTHING:		"none",
	C.GI_TRANSFER_CONTAINER:		"container",
	C.GI_TRANSFER_EVERYTHING:		"full",
}

var scopes = map[C.GIScopeType]string{
	C.GI_SCOPE_TYPE_INVALID:		"invalid",
	C.GI_SCOPE_TYPE_CALL:			"call",
	C.GI_SCOPE_TYPE_ASYNC:		"async",
	C.GI_SCOPE_TYPE_NOTIFIED:		"notified",
}

func arg(info *C.GIBaseInfo) {
	arg := (*C.GIArgInfo)(unsafe.Pointer(info))
	fmt.Printf("	closure = %d,\n", C.g_arg_info_get_closure(arg))
	fmt.Printf("	destroy = %d,\n", C.g_arg_info_get_destroy(arg))
	fmt.Printf("	direction = %s\n", directions[C.g_arg_info_get_direction(arg)])
	fmt.Printf("	transfer = %s\n", transfers[C.g_arg_info_get_ownership_transfer(arg)])
	fmt.Printf("	scope = %s\n", scopes[C.g_arg_info_get_scope(arg)])
	fmt.Printf("	type = ")
	xtype(C.g_arg_info_get_type(arg))
	fmt.Printf("\n")
	fmt.Printf("	allows-null = %s\n", fromgbool(C.g_arg_info_may_be_null(arg)))
	fmt.Printf("	caller-allocates = %s\n", fromgbool(C.g_arg_info_is_caller_allocates(arg)))
	fmt.Printf("	optional = %s\n", fromgbool(C.g_arg_info_is_optional(arg)))
	fmt.Printf("	is-return-value = %s\n", fromgbool(C.g_arg_info_is_return_value(arg)))
	fmt.Printf("	only-useful-for-C = %s\n", fromgbool(C.g_arg_info_is_skip(arg)))
}

func callable(info *C.GIBaseInfo) {
	call := (*C.GICallableInfo)(unsafe.Pointer(info))
	fmt.Printf("	can-return-gerror = %s\n", fromgbool(C.g_callable_info_can_throw_gerror(call)))
	fmt.Printf("	nargs = %d\n", C.g_callable_info_get_n_args(call))
	// TODO args
	fmt.Printf("	return-transfer = %s\n", transfers[C.g_callable_info_get_caller_owns(call)])
	fmt.Printf("	return-type = ")
	xtype(C.g_callable_info_get_return_type(call))
	fmt.Printf("\n")
	fmt.Printf("	is-method = %s\n", fromgbool(C.g_callable_info_is_method(call)))
	// TODO return attributes
	fmt.Printf("	can-return-null = %s\n", fromgbool(C.g_callable_info_may_return_null(call)))
	fmt.Printf("	return-only-useful-for-C = %s\n", fromgbool(C.g_callable_info_skip_return(call)))
}

func funcflags(flags C.GIFunctionInfoFlags) string {
	s := ""
	if (flags & C.GI_FUNCTION_IS_METHOD) != 0 {
		s += "| method"
	}
	if (flags & C.GI_FUNCTION_IS_CONSTRUCTOR) != 0 {
		s += "| constructor"
	}
	if (flags & C.GI_FUNCTION_IS_GETTER) != 0 {
		s += "| getter"
	}
	if (flags & C.GI_FUNCTION_IS_SETTER) != 0 {
		s += "| setter"
	}
	if (flags & C.GI_FUNCTION_WRAPS_VFUNC) != 0 {
		s += "| wrapsvfunc"
	}
	if (flags & C.GI_FUNCTION_THROWS) != 0 {
		s += "| throws"
	}
	if s == "" {
		return s
	}
	return s[2:]		// strip leading OR operator
}

func function(info *C.GIBaseInfo) {
	callable(info)
	f := (*C.GIFunctionInfo)(unsafe.Pointer(info))
	fmt.Printf("	flags = %s\n", funcflags(C.g_function_info_get_flags(f)))
	// TODO property
	fmt.Printf("	symbol = %q\n", fromgstr(C.g_function_info_get_symbol(f)))
	// TODO vfunc
}

func sigflags(flags C.GSignalFlags) string {
	s := ""
	if (flags & C.G_SIGNAL_RUN_FIRST) != 0 {
		s += "| runfirst"
	}
	if (flags & C.G_SIGNAL_RUN_LAST) != 0 {
		s += "| runlast"
	}
	if (flags & C.G_SIGNAL_RUN_CLEANUP) != 0 {
		s += "| runcleanup"
	}
	if (flags & C.G_SIGNAL_NO_RECURSE) != 0 {
		s += "| norecurse"
	}
	if (flags & C.G_SIGNAL_DETAILED) != 0 {
		s += "| detailed"
	}
	if (flags & C.G_SIGNAL_ACTION) != 0 {
		s += "| action"
	}
	if (flags & C.G_SIGNAL_NO_HOOKS) != 0 {
		s += "| nohooks"
	}
	if (flags & C.G_SIGNAL_MUST_COLLECT) != 0 {
		s += "| mustcollect"
	}
	if (flags & C.G_SIGNAL_DEPRECATED) != 0 {
		s += "| deprecated"
	}
	if s == "" {
		return s
	}
	return s[2:]		// strip leading OR operator
}

func signal(info *C.GIBaseInfo) {
	callable(info)
	sig := (*C.GISignalInfo)(unsafe.Pointer(info))
	fmt.Printf("	flags = %s\n", sigflags(C.g_signal_info_get_flags(sig)))
	// TODO class closure
	fmt.Printf("	true-stops-emit = %s\n", fromgbool(C.g_signal_info_true_stops_emit(sig)))
}

func xtype(t *C.GITypeInfo) {
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
		fmt.Printf("%s %s {\n",
			typenames[C.g_base_info_get_type(info)],
			fromgstr(C.g_base_info_get_name(info)))
		f := typefuncs[C.g_base_info_get_type(info)]
		if f != nil {
			f(info)
		}
		fmt.Printf("} attribs { \n")
		for C.g_base_info_iterate_attributes(info, &iter, &name, &value) != C.FALSE {
			fmt.Printf("	%q = %q,\n", C.GoString(name), C.GoString(value))
		}
		fmt.Printf("}\n")
		C.g_base_info_unref(info);
	}
}
