// 22 june 2014
package main

import (
	"unsafe"
	"errors"
)

// #cgo pkg-config: gobject-introspection-1.0
// #include <girepository.h>
// #include <stdlib.h>
import "C"

// TODOs
// - change int used for pointers to some named type

type InfoType int
const (
	TypeInvalid InfoType = C.GI_INFO_TYPE_INVALID
	TypeFunction InfoType = C.GI_INFO_TYPE_FUNCTION
	TypeCallback InfoType = C.GI_INFO_TYPE_CALLBACK
	TypeStruct InfoType = C.GI_INFO_TYPE_STRUCT
	TypeBoxed InfoType = C.GI_INFO_TYPE_BOXED
	TypeEnum InfoType = C.GI_INFO_TYPE_ENUM
	TypeFlags InfoType = C.GI_INFO_TYPE_FLAGS
	TypeObject InfoType = C.GI_INFO_TYPE_OBJECT
	TypeInterface InfoType = C.GI_INFO_TYPE_INTERFACE
	TypeConstant InfoType = C.GI_INFO_TYPE_CONSTANT
	TypeInvalid0 InfoType = C.GI_INFO_TYPE_INVALID_0
	TypeUnion InfoType = C.GI_INFO_TYPE_UNION
	TypeValue InfoType = C.GI_INFO_TYPE_VALUE
	TypeSignal InfoType = C.GI_INFO_TYPE_SIGNAL
	TypeVFunc InfoType = C.GI_INFO_TYPE_VFUNC
	TypeProperty InfoType = C.GI_INFO_TYPE_PROPERTY
	TypeField InfoType = C.GI_INFO_TYPE_FIELD
	TypeArg InfoType = C.GI_INFO_TYPE_ARG
	TypeType InfoType = C.GI_INFO_TYPE_TYPE
	TypeUnresolved InfoType = C.GI_INFO_TYPE_UNRESOLVED
)

type reader struct {
	ns			*Namespace
	baseInfos		map[*C.GIBaseInfo]int
	unref		[]*C.GIBaseInfo
}

func newReader(ns *Namespace) (r *reader) {
	r = new(reader)
	r.ns = ns
	r.baseInfos = map[*C.GIBaseInfo]int{}
	r.unref = make([]*C.GIBaseInfo, 0, 65536)
	return r
}

func (r *reader) find(info *C.GIBaseInfo) (n int) {
	n = -1
	for i, v := range r.baseInfos {
		if C.g_base_info_equal(info, i) != C.FALSE {
			n = v
			break
		}
	}
	if n != -1 {
		r.baseInfos[info] = n		// add for next time
	}
	return n
}

func (r *reader) found(info *C.GIBaseInfo, n int) int {
	r.baseInfos[info] = n
	return n
}

func (r *reader) queueUnref(info *C.GIBaseInfo) {
	r.unref = append(r.unref, info)
}

func (r *reader) unrefAll() {
	for _, p := range r.unref {
		C.g_base_info_unref(p)
	}
	r.unref = nil		// collect the list
}

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

func fromgstr(str *C.gchar) string {
	return C.GoString((*C.char)(unsafe.Pointer(str)))
}

func fromgbool(b C.gboolean) bool {
	return b != C.FALSE
}

func (r *reader) readBaseInfo(info *C.GIBaseInfo, out *BaseInfo) int {
	toplevel := out == nil
	if toplevel {
		if n := r.find(info); n != -1 {
			return n
		}
	}

	var iter C.GIAttributeIter		// properly initializes
	var name, value *C.char

	if out == nil {
		out = &BaseInfo{}
	}
	out.Type = InfoType(C.g_base_info_get_type(info))
	// there's an unbroken case bug in gir that makes the following line abort on GITypeInfos
	// see https://bugzilla.gnome.org/show_bug.cgi?id=709456
	// thanks lazka in irc.gimp.net/#gtk+
	if out.Type != TypeType {
		out.Name = fromgstr(C.g_base_info_get_name(info))
	}
	if out.Type != TypeUnresolved {	// will cause asking for attributes to crash (for instance, in GObject.VaClosureMarshal)
		out.Attributes = map[string]string{}
		for C.g_base_info_iterate_attributes(info, &iter, &name, &value) != C.FALSE {
			out.Attributes[C.GoString(name)] = C.GoString(value)
		}
	}
	out.Deprecated = fromgbool(C.g_base_info_is_deprecated(info))
	if toplevel {
		r.ns.OtherBaseInfos = append(r.ns.OtherBaseInfos, *out)
		return r.found(info, len(r.ns.OtherBaseInfos) -1)
	}
	return -1
}

type Direction int
const (
	In Direction = C.GI_DIRECTION_IN
	Out Direction = C.GI_DIRECTION_OUT
	InOut Direction = C.GI_DIRECTION_INOUT
)

type Transfer int
const (
	None Transfer = C.GI_TRANSFER_NOTHING
	Container Transfer = C.GI_TRANSFER_CONTAINER
	Full Transfer = C.GI_TRANSFER_EVERYTHING
)

type ScopeType int
const (
	ScopeInvalid ScopeType = C.GI_SCOPE_TYPE_INVALID
	ScopeCall ScopeType = C.GI_SCOPE_TYPE_CALL
	ScopeAsync ScopeType = C.GI_SCOPE_TYPE_ASYNC
	ScopeNotified ScopeType = C.GI_SCOPE_TYPE_NOTIFIED
)

type ArgInfo struct {
	BaseInfo
	Closure			int
	Destroy			int
	Direction			Direction
	OwnershipTransfer	Transfer
	Scope			ScopeType
	Type				int
	MayBeNull		bool
	CallerAllocates		bool
	Optional			bool
	IsReturnValue		bool
	OnlyUsefulForC	bool
}

func (r *reader) readArgInfo(info *C.GIArgInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	out := ArgInfo{}
	r.readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	out.Closure = int(C.g_arg_info_get_closure(info))
	out.Destroy = int(C.g_arg_info_get_destroy(info))
	out.Direction = Direction(C.g_arg_info_get_direction(info))
	out.OwnershipTransfer = Transfer(C.g_arg_info_get_ownership_transfer(info))
	out.Scope = ScopeType(C.g_arg_info_get_scope(info))
	ti := C.g_arg_info_get_type(info)
	out.Type = r.readTypeInfo(ti)
	r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
	out.MayBeNull = fromgbool(C.g_arg_info_may_be_null(info))
	out.CallerAllocates = fromgbool(C.g_arg_info_is_caller_allocates(info))
	out.Optional = fromgbool(C.g_arg_info_is_optional(info))
	out.IsReturnValue = fromgbool(C.g_arg_info_is_return_value(info))
	out.OnlyUsefulForC = fromgbool(C.g_arg_info_is_skip(info))
	r.ns.Args = append(r.ns.Args, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Args) - 1)
}

type CallableInfo struct {
	BaseInfo
	CanThrowGError		bool
	Args					[]int
	ReturnTransfer			Transfer
	ReturnAttributes		map[string]string
	ReturnType			int
	IsMethod				bool
	MayReturnNull			bool
	ReturnOnlyUsefulForC	bool
}

func (r *reader) readCallableInfo(info *C.GICallableInfo, out *CallableInfo) int {
	toplevel := out == nil
	if toplevel {
		if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
			return n
		}
	}

	var iter C.GIAttributeIter		// properly initializes
	var name, value *C.char

	if out == nil {
		out = &CallableInfo{}
	}
	r.readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	out.CanThrowGError = fromgbool(C.g_callable_info_can_throw_gerror(info))
	n := int(C.g_callable_info_get_n_args(info))
	out.Args = make([]int, n)
	for i := 0; i < n; i++ {
		ai := C.g_callable_info_get_arg(info, C.gint(i))
		out.Args[i] = r.readArgInfo(ai)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(ai)))
	}
	out.ReturnTransfer = Transfer(C.g_callable_info_get_caller_owns(info))
	out.ReturnAttributes = map[string]string{}
	for C.g_callable_info_iterate_return_attributes(info, &iter, &name, &value) != C.FALSE {
		out.ReturnAttributes[C.GoString(name)] = C.GoString(value)
	}
	ti := C.g_callable_info_get_return_type(info)
	out.ReturnType = r.readTypeInfo(ti)
	r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
	out.IsMethod = fromgbool(C.g_callable_info_is_method(info))
	out.MayReturnNull = fromgbool(C.g_callable_info_may_return_null(info))
	out.ReturnOnlyUsefulForC = fromgbool(C.g_callable_info_skip_return(info))
	if toplevel {
		r.ns.Callbacks = append(r.ns.Callbacks, *out)
		return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Callbacks) - 1)
	}
	return -1		// irrelevant
}

type FunctionInfoFlags int
const (
	FunctionIsMethod FunctionInfoFlags = C.GI_FUNCTION_IS_METHOD
	FunctionIsConstructor FunctionInfoFlags = C.GI_FUNCTION_IS_CONSTRUCTOR
	FunctionIsGetter FunctionInfoFlags = C.GI_FUNCTION_IS_GETTER
	FunctionIsSetter FunctionInfoFlags = C.GI_FUNCTION_IS_SETTER
	FunctionWrapsVFunc FunctionInfoFlags = C.GI_FUNCTION_WRAPS_VFUNC
	FunctionThrows FunctionInfoFlags = C.GI_FUNCTION_THROWS
)

type FunctionInfo struct {
	CallableInfo
	Flags		FunctionInfoFlags
	Property		int
	Symbol		string
	VFunc		int
}

func (r *reader) readFunctionInfo(info *C.GIFunctionInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	out := FunctionInfo{}
	r.readCallableInfo((*C.GICallableInfo)(unsafe.Pointer(info)), &out.CallableInfo)
	out.Flags = FunctionInfoFlags(C.g_function_info_get_flags(info))
	out.Property = -1
	if (out.Flags & (FunctionIsGetter | FunctionIsSetter)) != 0 {
		pi := C.g_function_info_get_property(info)
		if pi != nil {
			out.Property = r.readPropertyInfo(pi)
			r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(pi)))
		}
	}
	out.Symbol = fromgstr(C.g_function_info_get_symbol(info))
	out.VFunc = -1
	if (out.Flags & FunctionWrapsVFunc) != 0 {
		vi := C.g_function_info_get_vfunc(info)
		if vi != nil {
			out.VFunc = r.readVFuncInfo(vi)
			r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(vi)))
		}
	}
	r.ns.Functions = append(r.ns.Functions, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Functions) - 1)
}

// Note: this is a GObject enum, not a GObject Introspection enum
type SignalFlags int
const (
	SignalRunFirst SignalFlags = C.G_SIGNAL_RUN_FIRST
	SignalRunLast SignalFlags = C.G_SIGNAL_RUN_LAST
	SignalRunCleanup SignalFlags = C.G_SIGNAL_RUN_CLEANUP
	SignalNoRecurse SignalFlags = C.G_SIGNAL_NO_RECURSE
	SignalDetailed SignalFlags = C.G_SIGNAL_DETAILED
	SignalAction SignalFlags = C.G_SIGNAL_ACTION
	SignalNoHooks SignalFlags = C.G_SIGNAL_NO_HOOKS
	SignalMustCollect SignalFlags = C.G_SIGNAL_MUST_COLLECT
	SignalDeprecated SignalFlags = C.G_SIGNAL_DEPRECATED
)

type SignalInfo struct {
	CallableInfo
	Flags			SignalFlags
	ClassClosure		int
	TrueStopsEmit		bool
}

func (r *reader) readSignalInfo(info *C.GISignalInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	out := SignalInfo{}
	r.readCallableInfo((*C.GICallableInfo)(unsafe.Pointer(info)), &out.CallableInfo)
	out.Flags = SignalFlags(C.g_signal_info_get_flags(info))
	out.ClassClosure = -1
	vi := C.g_signal_info_get_class_closure(info)
	if vi != nil {
		out.ClassClosure = r.readVFuncInfo(vi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(vi)))
	}
	out.TrueStopsEmit = fromgbool(C.g_signal_info_true_stops_emit(info))
	r.ns.Signals = append(r.ns.Signals, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Signals) - 1)
}

type VFuncInfoFlags int
const (
	VFuncMustChainUp VFuncInfoFlags = C.GI_VFUNC_MUST_CHAIN_UP
	VFuncMustOverride VFuncInfoFlags = C.GI_VFUNC_MUST_OVERRIDE
	VFuncMustNotOverride VFuncInfoFlags = C.GI_VFUNC_MUST_NOT_OVERRIDE
	VFuncThrows VFuncInfoFlags = C.GI_VFUNC_THROWS
)

type VFuncInfo struct {
	CallableInfo
	Flags		VFuncInfoFlags
	Offset		int
	Signal		int
	Invoker		int
	// skip Address; that requires a GType
}

func (r *reader) readVFuncInfo(info *C.GIVFuncInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	out := VFuncInfo{}
	r.readCallableInfo((*C.GICallableInfo)(unsafe.Pointer(info)), &out.CallableInfo)
	out.Flags = VFuncInfoFlags(C.g_vfunc_info_get_flags(info))
	out.Offset = int(C.g_vfunc_info_get_offset(info))
	out.Signal = -1
	si := C.g_vfunc_info_get_signal(info)
	if si != nil {
		out.Signal = r.readSignalInfo(si)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(si)))
	}
	out.Invoker = -1
	fi := C.g_vfunc_info_get_invoker(info)
	if si != nil {
		out.Invoker = r.readFunctionInfo(fi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	// skip Address; that requires a GType
	r.ns.VFuncs = append(r.ns.VFuncs, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.VFuncs) - 1)
}

type ConstantInfo struct {
	BaseInfo
	Type			int
	Value		[]byte		// assume this is little-endian for now
	StringValue	string
}

func (r *reader) readConstantInfo(info *C.GIConstantInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	var value C.GIArgument

	out := ConstantInfo{}
	r.readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	ti := C.g_constant_info_get_type(info)
	out.Type = r.readTypeInfo(ti)
	r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
	n := C.g_constant_info_get_value(info, &value)
	// TODO string, pointer
	out.Value = make([]byte, n)
	copy(out.Value, value[:])
	C.g_constant_info_free_value(info, &value)
	r.ns.Constants = append(r.ns.Constants, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Constants) - 1)
}

type FieldInfoFlags int
const (
	FieldIsReadable FieldInfoFlags = C.GI_FIELD_IS_READABLE
	FieldIsWritable FieldInfoFlags = C.GI_FIELD_IS_WRITABLE
)

type FieldInfo struct {
	BaseInfo
	Flags		FieldInfoFlags
	Offset		int
	Size			int
	Type			int
}

func (r *reader) readFieldInfo(info *C.GIFieldInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	out := FieldInfo{}
	r.readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	out.Flags = FieldInfoFlags(C.g_field_info_get_flags(info))
	out.Offset = int(C.g_field_info_get_offset(info))
	out.Size = int(C.g_field_info_get_size(info))
	ti := C.g_field_info_get_type(info)
	out.Type = r.readTypeInfo(ti)
	r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
	r.ns.Fields = append(r.ns.Fields, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Fields) - 1)
}

// Note: this is a GObject enum, not a GObject Introspection enum
type ParamFlags int
const (
	ParamReadable ParamFlags = C.G_PARAM_READABLE
	ParamWritable ParamFlags = C.G_PARAM_WRITABLE
	ParamReadWrite ParamFlags = C.G_PARAM_READWRITE
	ParamConstruct ParamFlags = C.G_PARAM_CONSTRUCT
	ParamConstructOnly ParamFlags = C.G_PARAM_CONSTRUCT_ONLY
	ParamLaxValidation ParamFlags = C.G_PARAM_LAX_VALIDATION
	ParamStaticName ParamFlags = C.G_PARAM_STATIC_NAME
	ParamPrivate ParamFlags = C.G_PARAM_PRIVATE
	ParamStaticNick ParamFlags = C.G_PARAM_STATIC_NICK
	ParamStaticBlurb ParamFlags = C.G_PARAM_STATIC_BLURB
	ParamDeprecated ParamFlags = C.G_PARAM_DEPRECATED
	ParamStaticStrings ParamFlags = C.G_PARAM_STATIC_STRINGS
)

type PropertyInfo struct {
	BaseInfo
	Flags		ParamFlags
	Transfer		Transfer
	Type			int
}

func (r *reader) readPropertyInfo(info *C.GIPropertyInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	out := PropertyInfo{}
	r.readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	out.Flags = ParamFlags(C.g_property_info_get_flags(info))
	out.Transfer = Transfer(C.g_property_info_get_ownership_transfer(info))
	ti := C.g_property_info_get_type(info)
	out.Type = r.readTypeInfo(ti)
	r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
	r.ns.Properties = append(r.ns.Properties, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Properties) - 1)
}

type RegisteredTypeInfo struct {
	BaseInfo
	Name		string
	Init			string
	// skip GType (see below)
}

func readRegisteredTypeInfo(info *C.GIRegisteredTypeInfo, out *RegisteredTypeInfo) {
	// TODO
	newReader(nil).readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	out.Name = fromgstr(C.g_registered_type_info_get_type_name(info))
	out.Init = fromgstr(C.g_registered_type_info_get_type_init(info))
	// skip GType; we won't need it (and it causes problems with, for instance, GstPbutils) (also thanks to tristan in irc.gimp.net/#gtk+ for more information)
}

type TypeTag int
const (
	TagVoid TypeTag = C.GI_TYPE_TAG_VOID
	TagBoolean TypeTag = C.GI_TYPE_TAG_BOOLEAN
	TagInt8 TypeTag = C.GI_TYPE_TAG_INT8
	TagUint8 TypeTag = C.GI_TYPE_TAG_UINT8
	TagInt16 TypeTag = C.GI_TYPE_TAG_INT16
	TagUint16 TypeTag = C.GI_TYPE_TAG_UINT16
	TagInt32 TypeTag = C.GI_TYPE_TAG_INT32
	TagUint32 TypeTag = C.GI_TYPE_TAG_UINT32
	TagInt64 TypeTag = C.GI_TYPE_TAG_INT64
	TagUint64 TypeTag = C.GI_TYPE_TAG_UINT64
	TagFloat TypeTag = C.GI_TYPE_TAG_FLOAT
	TagDouble TypeTag = C.GI_TYPE_TAG_DOUBLE
	TagGtype TypeTag = C.GI_TYPE_TAG_GTYPE
	TagUtf8 TypeTag = C.GI_TYPE_TAG_UTF8
	TagFilename TypeTag = C.GI_TYPE_TAG_FILENAME
	TagArray TypeTag = C.GI_TYPE_TAG_ARRAY
	TagInterface TypeTag = C.GI_TYPE_TAG_INTERFACE
	TagGList TypeTag = C.GI_TYPE_TAG_GLIST
	TagGSList TypeTag = C.GI_TYPE_TAG_GSLIST
	TagGHashTable TypeTag = C.GI_TYPE_TAG_GHASH
	TagGError TypeTag = C.GI_TYPE_TAG_ERROR
	TagUnichar TypeTag = C.GI_TYPE_TAG_UNICHAR
)

type EnumInfo struct {
	RegisteredTypeInfo
	Values				[]int64
	ValuesInvalid			[]bool
	Methods				[]int
	StorageType			TypeTag
	ErrorDomain			string
}

func (r *reader) readEnumInfo(info *C.GIEnumInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	out := EnumInfo{}
	readRegisteredTypeInfo((*C.GIRegisteredTypeInfo)(unsafe.Pointer(info)), &out.RegisteredTypeInfo)
	n := int(C.g_enum_info_get_n_values(info))
	out.Values = make([]int64, n)
	out.ValuesInvalid = make([]bool, n)
	for i := 0; i < n; i++ {
		vi := C.g_enum_info_get_value(info, C.gint(n))
		if vi != nil {
			out.Values[i] = int64(C.g_value_info_get_value(vi))
			r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(vi)))
		} else {
			out.ValuesInvalid[i] = true
		}
	}
	n = int(C.g_enum_info_get_n_methods(info))
	out.Methods = make([]int, n)
	for i := 0; i < n; i++ {
		fi := C.g_enum_info_get_method(info, C.gint(i))
		out.Methods[i] = r.readFunctionInfo(fi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	out.StorageType = TypeTag(C.g_enum_info_get_storage_type(info))
	ed := C.g_enum_info_get_error_domain(info)
	if ed != nil {
		out.ErrorDomain = fromgstr(ed)
	}
	r.ns.Enums = append(r.ns.Enums, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Enums) - 1)
}

type InterfaceInfo struct {
	RegisteredTypeInfo
	Prerequisites			[]BaseInfo
	Properties				[]int
	Methods				[]int
	Signals				[]int
	VFuncs				[]int
	Constants				[]int
	Struct				int
}

func (r *reader) readInterfaceInfo(info *C.GIInterfaceInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	out := InterfaceInfo{}
	readRegisteredTypeInfo((*C.GIRegisteredTypeInfo)(unsafe.Pointer(info)), &out.RegisteredTypeInfo)
	n := int(C.g_interface_info_get_n_prerequisites(info))
	out.Prerequisites = make([]BaseInfo, n)
	for i := 0; i < n; i++ {
		bi := C.g_interface_info_get_prerequisite(info, C.gint(i))
		r.readBaseInfo(bi, &out.Prerequisites[i])
		r.queueUnref(bi)
	}
	n = int(C.g_interface_info_get_n_properties(info))
	out.Properties = make([]int, n)
	for i := 0; i < n; i++ {
		pi := C.g_interface_info_get_property(info, C.gint(i))
		out.Properties[i] = r.readPropertyInfo(pi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(pi)))
	}
	n = int(C.g_interface_info_get_n_methods(info))
	out.Methods = make([]int, n)
	for i := 0; i < n; i++ {
		fi := C.g_interface_info_get_method(info, C.gint(i))
		out.Methods[i] = r.readFunctionInfo(fi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	n = int(C.g_interface_info_get_n_signals(info))
	out.Signals = make([]int, n)
	for i := 0; i < n; i++ {
		si := C.g_interface_info_get_signal(info, C.gint(i))
		out.Signals[i] = r.readSignalInfo(si)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(si)))
	}
	n = int(C.g_interface_info_get_n_vfuncs(info))
	out.VFuncs = make([]int, n)
	for i := 0; i < n; i++ {
		vi := C.g_interface_info_get_vfunc(info, C.gint(i))
		out.VFuncs[i] = r.readVFuncInfo(vi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(vi)))
	}
	n = int(C.g_interface_info_get_n_constants(info))
	out.Constants = make([]int, n)
	for i := 0; i < n; i++ {
		ci := C.g_interface_info_get_constant(info, C.gint(i))
		out.Constants[i] = r.readConstantInfo(ci)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(ci)))
	}
	out.Struct = -1
	si := C.g_interface_info_get_iface_struct(info)
	if si != nil {
		out.Struct = r.readStructInfo(si)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(si)))
	}
	r.ns.Interfaces = append(r.ns.Interfaces, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Interfaces) - 1)
}

type ObjectInfo struct {
	RegisteredTypeInfo
	IsAbstract				bool
	IsFundamental			bool
	Parent				int
	Name				string
	Init					string
	Constants				[]int
	Fields				[]int
	Interfaces				[]int
	Methods				[]int
	Properties				[]int
	Signals				[]int
	VFuncs				[]int
	Struct				int
	RefFunction			string
	UnrefFunction			string
	SetValueFunction		string
	GetValueFunction		string
}

func (r *reader) readObjectInfo(info *C.GIObjectInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	out := ObjectInfo{}
	readRegisteredTypeInfo((*C.GIRegisteredTypeInfo)(unsafe.Pointer(info)), &out.RegisteredTypeInfo)
	out.IsAbstract = fromgbool(C.g_object_info_get_abstract(info))
	out.IsFundamental = fromgbool(C.g_object_info_get_fundamental(info))
	out.Parent = -1
	oi := C.g_object_info_get_parent(info)
	if oi != nil {
		out.Parent = r.readObjectInfo(oi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(oi)))
	}
	out.Name = fromgstr(C.g_object_info_get_type_name(info))
	out.Init = fromgstr(C.g_object_info_get_type_init(info))
	n := int(C.g_object_info_get_n_constants(info))
	out.Constants = make([]int, n)
	for i := 0; i < n; i++ {
		ci := C.g_object_info_get_constant(info, C.gint(i))
		out.Constants[i] = r.readConstantInfo(ci)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(ci)))
	}
	n = int(C.g_object_info_get_n_fields(info))
	out.Fields = make([]int, n)
	for i := 0; i < n; i++ {
		fi := C.g_object_info_get_field(info, C.gint(i))
		out.Fields[i] = r.readFieldInfo(fi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	n = int(C.g_object_info_get_n_interfaces(info))
	out.Interfaces = make([]int, n)
	for i := 0; i < n; i++ {
		ii := C.g_object_info_get_interface(info, C.gint(i))
		out.Interfaces[i] = r.readInterfaceInfo(ii)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(ii)))
	}
	n = int(C.g_object_info_get_n_methods(info))
	out.Methods = make([]int, n)
	for i := 0; i < n; i++ {
		fi := C.g_object_info_get_method(info, C.gint(i))
		out.Methods[i] = r.readFunctionInfo(fi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	n = int(C.g_object_info_get_n_properties(info))
	out.Properties = make([]int, n)
	for i := 0; i < n; i++ {
		pi := C.g_object_info_get_property(info, C.gint(i))
		out.Properties[i] = r.readPropertyInfo(pi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(pi)))
	}
	n = int(C.g_object_info_get_n_signals(info))
	out.Signals = make([]int, n)
	for i := 0; i < n; i++ {
		si := C.g_object_info_get_signal(info, C.gint(i))
		out.Signals[i] = r.readSignalInfo(si)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(si)))
	}
	n = int(C.g_object_info_get_n_vfuncs(info))
	out.VFuncs = make([]int, n)
	for i := 0; i < n; i++ {
		vi := C.g_object_info_get_vfunc(info, C.gint(i))
		out.VFuncs[i] = r.readVFuncInfo(vi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(vi)))
	}
	out.Struct = -1
	si := C.g_object_info_get_class_struct(info)
	if si != nil {
		out.Struct = r.readStructInfo(si)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(si)))
	}
	out.RefFunction = C.GoString(C.g_object_info_get_ref_function(info))
	out.UnrefFunction = C.GoString(C.g_object_info_get_unref_function(info))
	out.SetValueFunction = C.GoString(C.g_object_info_get_set_value_function(info))
	out.GetValueFunction = C.GoString(C.g_object_info_get_get_value_function(info))
	r.ns.Objects = append(r.ns.Objects, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Objects) - 1)
}

type StructInfo struct {
	RegisteredTypeInfo
	Alignment			uintptr
	Size					uintptr
	IsClassStruct			bool
	Foreign				bool
	Fields				[]int
	Methods				[]int
}

func (r *reader) readStructInfo(info *C.GIStructInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	out := StructInfo{}
	readRegisteredTypeInfo((*C.GIRegisteredTypeInfo)(unsafe.Pointer(info)), &out.RegisteredTypeInfo)
	out.Alignment = uintptr(C.g_struct_info_get_alignment(info))
	out.Size = uintptr(C.g_struct_info_get_size(info))
	out.IsClassStruct = fromgbool(C.g_struct_info_is_gtype_struct(info))
	out.Foreign = fromgbool(C.g_struct_info_is_foreign(info))
	n := int(C.g_struct_info_get_n_fields(info))
	out.Fields = make([]int, n)
	for i := 0; i < n; i++ {
		fi := C.g_struct_info_get_field(info, C.gint(i))
		out.Fields[i] = r.readFieldInfo(fi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	n = int(C.g_struct_info_get_n_methods(info))
	out.Methods = make([]int, n)
	for i := 0; i < n; i++ {
		fi := C.g_struct_info_get_method(info, C.gint(i))
		out.Methods[i] = r.readFunctionInfo(fi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	r.ns.Structs = append(r.ns.Structs, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Structs) - 1)
}

type UnionInfo struct {
	RegisteredTypeInfo
	Fields				[]int
	Methods				[]int
	Discriminated			bool
	DiscriminatorOffset		int
	DiscriminatorType		int
	DiscriminatorValues		[]int
	Size					uintptr
	Alignment			uintptr
}

func (r *reader) readUnionInfo(info *C.GIUnionInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	out := UnionInfo{}
	readRegisteredTypeInfo((*C.GIRegisteredTypeInfo)(unsafe.Pointer(info)), &out.RegisteredTypeInfo)
	n := int(C.g_union_info_get_n_fields(info))
	out.Fields = make([]int, n)
	out.DiscriminatorValues = make([]int, n)
	for i := 0; i < n; i++ {
		fi := C.g_union_info_get_field(info, C.gint(i))
		out.Fields[i] = r.readFieldInfo(fi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
		// do discriminator values here too
		out.DiscriminatorValues[i] = -1
		ci := C.g_union_info_get_discriminator(info, C.gint(i))
		if ci != nil {		// TODO this should probably just be a skip of the whole thing if there is no discriminator but meh
			out.DiscriminatorValues[i] = r.readConstantInfo(ci)
			r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(ci)))
		}
	}
	n = int(C.g_union_info_get_n_methods(info))
	out.Methods = make([]int, n)
	for i := 0; i < n; i++ {
		fi := C.g_union_info_get_method(info, C.gint(i))
		out.Methods[i] = r.readFunctionInfo(fi)
		r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	out.Discriminated = fromgbool(C.g_union_info_is_discriminated(info))
	out.DiscriminatorOffset = int(C.g_union_info_get_discriminator_offset(info))
	ti := C.g_union_info_get_discriminator_type(info)
	out.DiscriminatorType = r.readTypeInfo(ti)
	r.queueUnref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
	// discriminator values handled above
	out.Size = uintptr(C.g_union_info_get_size(info))
	out.Alignment = uintptr(C.g_union_info_get_alignment(info))
	r.ns.Unions = append(r.ns.Unions, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Unions) - 1)
}

type ArrayType int
const (
	CArray ArrayType = C.GI_ARRAY_TYPE_C
	GArray ArrayType = C.GI_ARRAY_TYPE_ARRAY
	GPtrArray ArrayType = C.GI_ARRAY_TYPE_PTR_ARRAY
	GByteArray ArrayType = C.GI_ARRAY_TYPE_BYTE_ARRAY
)

type TypeInfo struct {
	BaseInfo
	IsPointer			bool
	Tag				TypeTag
	// TODO ParamTypes
	Interface			BaseInfo
	ArrayLength		int
	ArrayFixedSize		int
	IsZeroTerminated	bool
	ArrayType		ArrayType
}

func (r *reader) readTypeInfo(info *C.GITypeInfo) int {
	if n := r.find((*C.GIBaseInfo)(unsafe.Pointer(info))); n != -1 {
		return n
	}

	out := TypeInfo{}
	r.readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	out.IsPointer = fromgbool(C.g_type_info_is_pointer(info))
	out.Tag = TypeTag(C.g_type_info_get_tag(info))
	// TODO ParamTypes
	bi := C.g_type_info_get_interface(info)
	if bi != nil {
		r.readBaseInfo(bi, &out.Interface)
		r.queueUnref(bi)
	}
	if out.Tag == TagArray {
		out.ArrayLength = int(C.g_type_info_get_array_length(info))
		out.ArrayFixedSize = int(C.g_type_info_get_array_fixed_size(info))
		out.IsZeroTerminated = fromgbool(C.g_type_info_is_zero_terminated(info))
		out.ArrayType = ArrayType(C.g_type_info_get_array_type(info))
	}
	r.ns.Types = append(r.ns.Types, out)
	return r.found((*C.GIBaseInfo)(unsafe.Pointer(info)), len(r.ns.Types) - 1)
}

type Namespace struct {
	// TODO other fields
	OtherBaseInfos		[]BaseInfo		// invalids, invalid0s, unresolveds
	// Invalids in OtherBaseInfos
	Functions			[]FunctionInfo
	Callbacks			[]CallableInfo
	Structs			[]StructInfo
	// TODO Boxed
	Enums			[]EnumInfo
	// TODO Flags
	Objects			[]ObjectInfo
	Interfaces			[]InterfaceInfo
	Constants			[]ConstantInfo
	// Invalid0s in OtherBaseInfos
	Unions			[]UnionInfo
	// TODO values
	Signals			[]SignalInfo
	VFuncs			[]VFuncInfo
	Properties			[]PropertyInfo
	Fields			[]FieldInfo
	Args				[]ArgInfo
	Types			[]TypeInfo
	// Unresolveds in OtherBaseInfos

	TopLevelInvalids		[]int
	TopLevelFunctions		[]int
	TopLevelCallbacks		[]int
	TopLevelStructs		[]int
	TopLevelBoxeds		[]int
	TopLevelEnums		[]int
	TopLevelFlags			[]int
	TopLevelObjects		[]int
	TopLevelInterfaces		[]int
	TopLevelConstants		[]int
	TopLevelInvalid0s		[]int
	TopLevelUnions		[]int
	// TODO values
	TopLevelSignals		[]int
	TopLevelVFuncs		[]int
	TopLevelProperties		[]int
	TopLevelFields			[]int
	TopLevelArgs			[]int
	TopLevelTypes			[]int
	TopLevelUnresolveds	[]int

}

func ReadNamespace(nsname string, version string) (ns Namespace, err error) {
	var gerr *C.GError = nil
	var cver *C.gchar = nil

	cns := (*C.gchar)(unsafe.Pointer(C.CString(nsname)))
	defer C.free(unsafe.Pointer(cns))
	if version != "" {
		cver := (*C.gchar)(unsafe.Pointer(C.CString(version)))
		defer C.free(unsafe.Pointer(cver))
	}
	if C.g_irepository_require(nil, cns, cver, 0, &gerr) == nil {
		return Namespace{}, errors.New(fromgstr(gerr.message))	// TODO adorn
	}
	n := int(C.g_irepository_get_n_infos(nil, cns))
	r := newReader(&ns)
	for i := 0; i < n; i++ {
		info := C.g_irepository_get_info(nil, cns, C.gint(i))
		switch InfoType(C.g_base_info_get_type(info)) {
		case TypeInvalid:
			ns.TopLevelInvalids = append(ns.TopLevelInvalids, r.readBaseInfo(info, nil))
		case TypeFunction:
			ns.TopLevelFunctions = append(ns.TopLevelFunctions, r.readFunctionInfo((*C.GIFunctionInfo)(unsafe.Pointer(info))))
		case TypeCallback:
			ns.TopLevelCallbacks = append(ns.TopLevelCallbacks, r.readCallableInfo((*C.GICallableInfo)(unsafe.Pointer(info)), nil))
		case TypeStruct:
			ns.TopLevelStructs = append(ns.TopLevelStructs, r.readStructInfo((*C.GIStructInfo)(unsafe.Pointer(info))))
		case TypeBoxed:
			// TODO
		case TypeEnum:
			ns.TopLevelEnums = append(ns.TopLevelEnums, r.readEnumInfo((*C.GIEnumInfo)(unsafe.Pointer(info))))
		case TypeFlags:
			// TODO
		case TypeObject:
			ns.TopLevelObjects = append(ns.TopLevelObjects, r.readObjectInfo((*C.GIObjectInfo)(unsafe.Pointer(info))))
		case TypeInterface:
			ns.TopLevelInterfaces = append(ns.TopLevelInterfaces, r.readInterfaceInfo((*C.GIInterfaceInfo)(unsafe.Pointer(info))))
		case TypeConstant:
			ns.TopLevelConstants = append(ns.TopLevelConstants, r.readConstantInfo((*C.GIConstantInfo)(unsafe.Pointer(info))))
		case TypeInvalid0:
			ns.TopLevelInvalid0s = append(ns.TopLevelInvalid0s, r.readBaseInfo(info, nil))
		case TypeUnion:
			ns.TopLevelUnions = append(ns.TopLevelUnions, r.readUnionInfo((*C.GIUnionInfo)(unsafe.Pointer(info))))
		case TypeValue:
			// TODO
		case TypeSignal:
			ns.TopLevelSignals = append(ns.TopLevelSignals, r.readSignalInfo((*C.GISignalInfo)(unsafe.Pointer(info))))
		case TypeVFunc:
			ns.TopLevelVFuncs = append(ns.TopLevelVFuncs, r.readVFuncInfo((*C.GIVFuncInfo)(unsafe.Pointer(info))))
		case TypeProperty:
			ns.TopLevelProperties = append(ns.TopLevelProperties, r.readPropertyInfo((*C.GIPropertyInfo)(unsafe.Pointer(info))))
		case TypeField:
			ns.TopLevelFields = append(ns.TopLevelFields, r.readFieldInfo((*C.GIFieldInfo)(unsafe.Pointer(info))))
		case TypeArg:
			ns.TopLevelArgs = append(ns.TopLevelArgs, r.readArgInfo((*C.GIArgInfo)(unsafe.Pointer(info))))
		case TypeType:
			ns.TopLevelTypes = append(ns.TopLevelTypes, r.readTypeInfo((*C.GITypeInfo)(unsafe.Pointer(info))))
		case TypeUnresolved:
			ns.TopLevelUnresolveds = append(ns.TopLevelUnresolveds, r.readBaseInfo(info, nil))
		default:
			panic("unknown info type")
		}
		r.queueUnref(info)
	}
	r.unrefAll()
	return ns, nil
}
