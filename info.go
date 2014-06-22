// 22 june 2014
package main

type InfoType int
const (
  GI_INFO_TYPE_INVALID,
  GI_INFO_TYPE_FUNCTION,
  GI_INFO_TYPE_CALLBACK,
  GI_INFO_TYPE_STRUCT,
  GI_INFO_TYPE_BOXED,
  GI_INFO_TYPE_ENUM,         /*  5 */
  GI_INFO_TYPE_FLAGS,
  GI_INFO_TYPE_OBJECT,
  GI_INFO_TYPE_INTERFACE,
  GI_INFO_TYPE_CONSTANT,
  GI_INFO_TYPE_INVALID_0,    /* 10 */
  GI_INFO_TYPE_UNION,
  GI_INFO_TYPE_VALUE,
  GI_INFO_TYPE_SIGNAL,
  GI_INFO_TYPE_VFUNC,
  GI_INFO_TYPE_PROPERTY,     /* 15 */
  GI_INFO_TYPE_FIELD,
  GI_INFO_TYPE_ARG,
  GI_INFO_TYPE_TYPE,
  GI_INFO_TYPE_UNRESOLVED
)

type BaseInfo struct {
	Type			InfoType
	Name		string
	Attributes		map[string]string
	Deprecated	bool
}

func (BaseInfo) baseInfo() {}

type Info interface {
	baseInfo()
}

// TODO fromgstr

func fromgbool(b C.gboolean) bool {
	return b != C.FALSE
}

func readBaseInfo(info *C.GIBaseInfo, out *BaseInfo) {
	var iter C.GIAttributeIter		// properly initializes
	var name, value *C.char

	out.Type = InfoType(C.g_base_info_get_type(info))
	out.Name = fromgstr(C.g_base_info_get_name(info))
	out.Attributes = map[string]string{}
	for C.g_base_info_iterate_attributes(info, &iter, &name, &value) != C.FALSE {
		out.Attributes[C.GoString(name)] = C.GoString(value)
	}
	out.Deprecated = fromgbool(C.g_type_info_is_deprecated(info))
}

type Direction int
const (
  GI_DIRECTION_IN,
  GI_DIRECTION_OUT,
  GI_DIRECTION_INOUT
)

type Transfer int
const (
  GI_TRANSFER_NOTHING,
  GI_TRANSFER_CONTAINER,
  GI_TRANSFER_EVERYTHING
)

type ScopeType int
const (
  GI_SCOPE_TYPE_INVALID,
  GI_SCOPE_TYPE_CALL,
  GI_SCOPE_TYPE_ASYNC,
  GI_SCOPE_TYPE_NOTIFIED
)

type ArgInfo struct {
	BaseInfo
	Closure			int
	Destroy			int
	Direction			Direction
	OwnershipTransfer	Transfer
	Scope			ScopeType
	Type				TypeInfo
	MayBeNull		bool
	CallerAllocates		bool
	Optional			bool
	IsReturnValue		bool
	OnlyUsefulForC	bool
}

func readArgInfo(info *C.GIArgInfo, out *ArgInfo) {
	readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	out.Closure = int(C.g_arg_info_get_closure(info))
	out.Destroy = int(C.g_arg_info_get_destroy(info))
	out.Direction = Direction(C.g_arg_info_get_direction(info))
	out.OwnershipTransfer = Transfer(C.g_arg_info_get_ownership_transfer(info))
	out.ScopeType = ScopeType(C.g_arg_info_get_scope(info))
	ti := C.g_arg_info_get_type(info)
	readTypeInfo(ti, &out.Type)
	C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
	out.MayBeNull = fromgbool(C.g_arg_info_may_be_null(info))
	out.CallerAllocates = fromgbool(C.g_arg_info_is_caller_allocates(info))
	out.Optional = fromgbol(C.g_arg_info_is_optional(info))
	out.IsReturnValue = fromgbool(C.g_arg_info_is_return_value(info))
	out.OnlyUsefulForC = fromgbool(C.g_arg_info_is_skip(info))
}

type CallableInfo struct {
	BaseInfo
	CanThrowGError		bool
	Args					[]ArgInfo
	ReturnTransfer			Transfer
	ReturnAttributes		map[string]string
	ReturnType			TypeInfo
	IsMethod				bool
	MayReturnNull			bool
	ReturnOnlyUsefulForC	bool
}

func readCallableInfo(info *C.GICallableInfo, out *CallableInfo) {
	var iter C.GIAttributeIter		// properly initializes
	var name, value *C.char

	readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	out.CanThrowGError = fromgbool(C.g_callable_info_can_throw_gerror(info))
	n := int(C.g_callable_info_get_n_args(info))
	out.Args = make([]ArgInfo, n)
	for i := 0; i < n; i++ {
		ai := C.g_callable_info_get_arg(info, C.gint(i))
		readArgInfo(ai, &out.Args[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(ai)))
	}
	out.ReturnTransfer = Transfer(C.g_callable_info_get_caller_owns(info))
	out.ReturnAttributes = map[string]string{}
	for C.g_callable_info_iterate_return_attributes(info, &iter, &name, &value) != C.FALSE {
		out.ReturnAttributes[C.GoString(name)] = C.GoString(value)
	}
	ti := C.g_callable_info_get_return_type(info)
	readTypeInfo(ti, &out.ReturnType)
	C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
	out.IsMethod = fromgbool(C.g_callable_info_is_method(info))
	out.MayReturnNull = fromgbool(C.g_callable_info_may_return_null(info))
	out.ReturnOnlyUsefulForC = fromgbool(C.g_callable_info_skip_return(info))
}

type FunctionInfoFlags int
const (
  GI_FUNCTION_IS_METHOD      = 1 << 0,
  GI_FUNCTION_IS_CONSTRUCTOR = 1 << 1,
  GI_FUNCTION_IS_GETTER      = 1 << 2,
  GI_FUNCTION_IS_SETTER      = 1 << 3,
  GI_FUNCTION_WRAPS_VFUNC    = 1 << 4,
  GI_FUNCTION_THROWS         = 1 << 5
)

type FunctionInfo struct {
	CallableInfo
	Flags		FunctionInfoFlags
	Property		PropertyInfo
	Symbol		string
	VFunc		VFuncInfo
}

func readFunctionInfo(info *C.GIFunctionInfo, out *FunctionInfo) {
	readCallableInfo((*C.GICallableInfo)(unsafe.Pointer(info)), &out.CallableInfo)
	out.Flags = FunctionInfoFlags(C.g_function_info_get_flags(info))
	pi := C.g_function_info_get_property(info)
	if pi != nil {
		readPropertyInfo(pi, &out.Property)
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(pi)))
	}
	out.Symbol = fromgstr(C.g_function_info_get_symbol(info))
	vi := C.g_function_info_get_vfunc(info)
	if vi != nil {
		readVFuncInfo(vi, &out.VFunc)
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(vi)))
	}
}
