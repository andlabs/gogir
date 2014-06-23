// 22 june 2014
package main

import (
"os"
"encoding/json"
"io"
"bytes"
	"unsafe"
)

// #cgo pkg-config: gobject-introspection-1.0
// #include <girepository.h>
// #include <stdlib.h>
import "C"

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

func readBaseInfo(info *C.GIBaseInfo, out *BaseInfo) {
	var iter C.GIAttributeIter		// properly initializes
	var name, value *C.char

	out.Type = InfoType(C.g_base_info_get_type(info))
	// there's an unbroken case bug in gir that makes the following line abort on GITypeInfos
	// see https://bugzilla.gnome.org/show_bug.cgi?id=709456
	// thanks lazka in irc.gimp.net/#gtk+
	if out.Type != TypeType {
		out.Name = fromgstr(C.g_base_info_get_name(info))
	}
	out.Attributes = map[string]string{}
	for C.g_base_info_iterate_attributes(info, &iter, &name, &value) != C.FALSE {
		out.Attributes[C.GoString(name)] = C.GoString(value)
	}
	out.Deprecated = fromgbool(C.g_base_info_is_deprecated(info))
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
	out.Scope = ScopeType(C.g_arg_info_get_scope(info))
	ti := C.g_arg_info_get_type(info)
	readTypeInfo(ti, &out.Type)
	C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
	out.MayBeNull = fromgbool(C.g_arg_info_may_be_null(info))
	out.CallerAllocates = fromgbool(C.g_arg_info_is_caller_allocates(info))
	out.Optional = fromgbool(C.g_arg_info_is_optional(info))
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
	Property		PropertyInfo
	Symbol		string
	VFunc		VFuncInfo
}

func readFunctionInfo(info *C.GIFunctionInfo, out *FunctionInfo) {
	readCallableInfo((*C.GICallableInfo)(unsafe.Pointer(info)), &out.CallableInfo)
	out.Flags = FunctionInfoFlags(C.g_function_info_get_flags(info))
	if (out.Flags & (FunctionIsGetter | FunctionIsSetter)) != 0 {
		pi := C.g_function_info_get_property(info)
		if pi != nil {
			readPropertyInfo(pi, &out.Property)
			C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(pi)))
		}
	}
	out.Symbol = fromgstr(C.g_function_info_get_symbol(info))
	if (out.Flags & FunctionWrapsVFunc) != 0 {
		vi := C.g_function_info_get_vfunc(info)
		if vi != nil {
			readVFuncInfo(vi, &out.VFunc)
			C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(vi)))
		}
	}
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
	ClassClosure		VFuncInfo
	TrueStopsEmit		bool
}

func readSignalInfo(info *C.GISignalInfo, out *SignalInfo) {
	readCallableInfo((*C.GICallableInfo)(unsafe.Pointer(info)), &out.CallableInfo)
	out.Flags = SignalFlags(C.g_signal_info_get_flags(info))
	vi := C.g_signal_info_get_class_closure(info)
	if vi != nil {
		readVFuncInfo(vi, &out.ClassClosure)
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(vi)))
	}
	out.TrueStopsEmit = fromgbool(C.g_signal_info_true_stops_emit(info))
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
	Signal		*SignalInfo
	Invoker		*FunctionInfo
	// skip Address; that requires a GType
}

func readVFuncInfo(info *C.GIVFuncInfo, out  *VFuncInfo) {
	readCallableInfo((*C.GICallableInfo)(unsafe.Pointer(info)), &out.CallableInfo)
	out.Flags = VFuncInfoFlags(C.g_vfunc_info_get_flags(info))
	out.Offset = int(C.g_vfunc_info_get_offset(info))
	si := C.g_vfunc_info_get_signal(info)
	if si != nil {
		out.Signal = new(SignalInfo)
		readSignalInfo(si, out.Signal)
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(si)))
	}
	fi := C.g_vfunc_info_get_invoker(info)
	if si != nil {
		out.Invoker = new(FunctionInfo)
		readFunctionInfo(fi, out.Invoker)
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	// skip Address; that requires a GType
}

type ConstantInfo struct {
	BaseInfo
	Type			TypeInfo
	Value		[]byte		// assume this is little-endian for now
	StringValue	string
}

func readConstantInfo(info *C.GIConstantInfo, out *ConstantInfo) {
	var value C.GIArgument

	readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	ti := C.g_constant_info_get_type(info)
	readTypeInfo(ti, &out.Type)
	C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
	n := C.g_constant_info_get_value(info, &value)
	// TODO string, pointer
	out.Value = make([]byte, n)
	copy(out.Value, value[:])
	C.g_constant_info_free_value(info, &value)
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
	Type			TypeInfo
}

func readFieldInfo(info *C.GIFieldInfo, out *FieldInfo) {
	readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	out.Flags = FieldInfoFlags(C.g_field_info_get_flags(info))
	out.Offset = int(C.g_field_info_get_offset(info))
	out.Size = int(C.g_field_info_get_size(info))
	ti := C.g_field_info_get_type(info)
	readTypeInfo(ti, &out.Type)
	C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
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
	Type			TypeInfo
}

func readPropertyInfo(info *C.GIPropertyInfo, out *PropertyInfo) {
	readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	out.Flags = ParamFlags(C.g_property_info_get_flags(info))
	out.Transfer = Transfer(C.g_property_info_get_ownership_transfer(info))
	ti := C.g_property_info_get_type(info)
	readTypeInfo(ti, &out.Type)
	C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
}

// Note: this is a GObject type, not a GObject Introspection type
// TODO safe to assume signedness?
type GType uintptr

type RegisteredTypeInfo struct {
	BaseInfo
	Name		string
	Init			string
	GType		GType
}

func readRegisteredTypeInfo(info *C.GIRegisteredTypeInfo, out *RegisteredTypeInfo) {
	readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	out.Name = fromgstr(C.g_registered_type_info_get_type_name(info))
	out.Init = fromgstr(C.g_registered_type_info_get_type_init(info))
	out.GType = GType(C.g_registered_type_info_get_g_type(info))
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
	Methods				[]FunctionInfo
	StorageType			TypeTag
	ErrorDomain			string
}

func readEnumInfo(info *C.GIEnumInfo, out *EnumInfo) {
	readRegisteredTypeInfo((*C.GIRegisteredTypeInfo)(unsafe.Pointer(info)), &out.RegisteredTypeInfo)
	n := int(C.g_enum_info_get_n_values(info))
	out.Values = make([]int64, n)
	out.ValuesInvalid = make([]bool, n)
	for i := 0; i < n; i++ {
		vi := C.g_enum_info_get_value(info, C.gint(n))
		if vi != nil {
			out.Values[i] = int64(C.g_value_info_get_value(vi))
			C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(vi)))
		} else {
			out.ValuesInvalid[i] = true
		}
	}
	n = int(C.g_enum_info_get_n_methods(info))
	out.Methods = make([]FunctionInfo, n)
	for i := 0; i < n; i++ {
		fi := C.g_enum_info_get_method(info, C.gint(i))
		readFunctionInfo(fi, &out.Methods[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	out.StorageType = TypeTag(C.g_enum_info_get_storage_type(info))
	ed := C.g_enum_info_get_error_domain(info)
	if ed != nil {
		out.ErrorDomain = fromgstr(ed)
	}
}

type InterfaceInfo struct {
	RegisteredTypeInfo
	Prerequisites			[]BaseInfo
	Properties				[]PropertyInfo
	Methods				[]FunctionInfo
	Signals				[]SignalInfo
	VFuncs				[]VFuncInfo
	Constants				[]ConstantInfo
	Struct				StructInfo
}

func readInterfaceInfo(info *C.GIInterfaceInfo, out *InterfaceInfo) {
	readRegisteredTypeInfo((*C.GIRegisteredTypeInfo)(unsafe.Pointer(info)), &out.RegisteredTypeInfo)
	n := int(C.g_interface_info_get_n_prerequisites(info))
	out.Prerequisites = make([]BaseInfo, n)
	for i := 0; i < n; i++ {
		bi := C.g_interface_info_get_prerequisite(info, C.gint(i))
		readBaseInfo(bi, &out.Prerequisites[i])
		C.g_base_info_unref(bi)
	}
	n = int(C.g_interface_info_get_n_properties(info))
	out.Properties = make([]PropertyInfo, n)
	for i := 0; i < n; i++ {
		pi := C.g_interface_info_get_property(info, C.gint(i))
		readPropertyInfo(pi, &out.Properties[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(pi)))
	}
	n = int(C.g_interface_info_get_n_methods(info))
	out.Methods = make([]FunctionInfo, n)
	for i := 0; i < n; i++ {
		fi := C.g_interface_info_get_method(info, C.gint(i))
		readFunctionInfo(fi, &out.Methods[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	n = int(C.g_interface_info_get_n_signals(info))
	out.Signals = make([]SignalInfo, n)
	for i := 0; i < n; i++ {
		si := C.g_interface_info_get_signal(info, C.gint(i))
		readSignalInfo(si, &out.Signals[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(si)))
	}
	n = int(C.g_interface_info_get_n_vfuncs(info))
	out.VFuncs = make([]VFuncInfo, n)
	for i := 0; i < n; i++ {
		vi := C.g_interface_info_get_vfunc(info, C.gint(i))
		readVFuncInfo(vi, &out.VFuncs[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(vi)))
	}
	n = int(C.g_interface_info_get_n_constants(info))
	out.Constants = make([]ConstantInfo, n)
	for i := 0; i < n; i++ {
		ci := C.g_interface_info_get_constant(info, C.gint(i))
		readConstantInfo(ci, &out.Constants[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(ci)))
	}
	si := C.g_interface_info_get_iface_struct(info)
	if si != nil {
		readStructInfo(si, &out.Struct)
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(si)))
	}
}

type ObjectInfo struct {
	RegisteredTypeInfo
	IsAbstract				bool
	IsFundamental			bool
	Parent				*ObjectInfo
	Name				string
	Init					string
	Constants				[]ConstantInfo
	Fields				[]FieldInfo
	Interfaces				[]InterfaceInfo
	Methods				[]FunctionInfo
	Properties				[]PropertyInfo
	Signals				[]SignalInfo
	VFuncs				[]VFuncInfo
	Struct				StructInfo
	RefFunction			string
	UnrefFunction			string
	SetValueFunction		string
	GetValueFunction		string
}

func readObjectInfo(info *C.GIObjectInfo, out *ObjectInfo) {
	readRegisteredTypeInfo((*C.GIRegisteredTypeInfo)(unsafe.Pointer(info)), &out.RegisteredTypeInfo)
	out.IsAbstract = fromgbool(C.g_object_info_get_abstract(info))
	out.IsFundamental = fromgbool(C.g_object_info_get_fundamental(info))
	oi := C.g_object_info_get_parent(info)
	if oi != nil {
		out.Parent = new(ObjectInfo)
		readObjectInfo(oi, out.Parent)
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(oi)))
	}
	out.Name = fromgstr(C.g_object_info_get_type_name(info))
	out.Init = fromgstr(C.g_object_info_get_type_init(info))
	n := int(C.g_object_info_get_n_constants(info))
	out.Constants = make([]ConstantInfo, n)
	for i := 0; i < n; i++ {
		ci := C.g_object_info_get_constant(info, C.gint(i))
		readConstantInfo(ci, &out.Constants[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(ci)))
	}
	n = int(C.g_object_info_get_n_fields(info))
	out.Fields = make([]FieldInfo, n)
	for i := 0; i < n; i++ {
		fi := C.g_object_info_get_field(info, C.gint(i))
		readFieldInfo(fi, &out.Fields[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	n = int(C.g_object_info_get_n_interfaces(info))
	out.Interfaces = make([]InterfaceInfo, n)
	for i := 0; i < n; i++ {
		ii := C.g_object_info_get_interface(info, C.gint(i))
		readInterfaceInfo(ii, &out.Interfaces[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(ii)))
	}
	n = int(C.g_object_info_get_n_methods(info))
	out.Methods = make([]FunctionInfo, n)
	for i := 0; i < n; i++ {
		fi := C.g_object_info_get_method(info, C.gint(i))
		readFunctionInfo(fi, &out.Methods[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	n = int(C.g_object_info_get_n_properties(info))
	out.Properties = make([]PropertyInfo, n)
	for i := 0; i < n; i++ {
		pi := C.g_object_info_get_property(info, C.gint(i))
		readPropertyInfo(pi, &out.Properties[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(pi)))
	}
	n = int(C.g_object_info_get_n_signals(info))
	out.Signals = make([]SignalInfo, n)
	for i := 0; i < n; i++ {
		si := C.g_object_info_get_signal(info, C.gint(i))
		readSignalInfo(si, &out.Signals[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(si)))
	}
	n = int(C.g_object_info_get_n_vfuncs(info))
	out.VFuncs = make([]VFuncInfo, n)
	for i := 0; i < n; i++ {
		vi := C.g_object_info_get_vfunc(info, C.gint(i))
		readVFuncInfo(vi, &out.VFuncs[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(vi)))
	}
	si := C.g_object_info_get_class_struct(info)
	if si != nil {
		readStructInfo(si, &out.Struct)
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(si)))
	}
	out.RefFunction = C.GoString(C.g_object_info_get_ref_function(info))
	out.UnrefFunction = C.GoString(C.g_object_info_get_unref_function(info))
	out.SetValueFunction = C.GoString(C.g_object_info_get_set_value_function(info))
	out.GetValueFunction = C.GoString(C.g_object_info_get_get_value_function(info))
}

type StructInfo struct {
	RegisteredTypeInfo
	Alignment			uintptr
	Size					uintptr
	IsClassStruct			bool
	Foreign				bool
	Fields				[]FieldInfo
	Methods				[]FunctionInfo
}

func readStructInfo(info *C.GIStructInfo, out *StructInfo) {
	readRegisteredTypeInfo((*C.GIRegisteredTypeInfo)(unsafe.Pointer(info)), &out.RegisteredTypeInfo)
	out.Alignment = uintptr(C.g_struct_info_get_alignment(info))
	out.Size = uintptr(C.g_struct_info_get_size(info))
	out.IsClassStruct = fromgbool(C.g_struct_info_is_gtype_struct(info))
	out.Foreign = fromgbool(C.g_struct_info_is_foreign(info))
	n := int(C.g_struct_info_get_n_fields(info))
	out.Fields = make([]FieldInfo, n)
	for i := 0; i < n; i++ {
		fi := C.g_struct_info_get_field(info, C.gint(i))
		readFieldInfo(fi, &out.Fields[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	n = int(C.g_struct_info_get_n_methods(info))
	out.Methods = make([]FunctionInfo, n)
	for i := 0; i < n; i++ {
		fi := C.g_struct_info_get_method(info, C.gint(i))
		readFunctionInfo(fi, &out.Methods[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
}

type UnionInfo struct {
	RegisteredTypeInfo
	Fields				[]FieldInfo
	Methods				[]FunctionInfo
	Discriminated			bool
	DiscriminatorOffset		int
	DiscriminatorType		TypeInfo
	DiscriminatorValues		[]ConstantInfo
	Size					uintptr
	Alignment			uintptr
}

func readUnionInfo(info *C.GIUnionInfo, out *UnionInfo) {
	readRegisteredTypeInfo((*C.GIRegisteredTypeInfo)(unsafe.Pointer(info)), &out.RegisteredTypeInfo)
	n := int(C.g_union_info_get_n_fields(info))
	out.Fields = make([]FieldInfo, n)
	out.DiscriminatorValues = make([]ConstantInfo, n)
	for i := 0; i < n; i++ {
		fi := C.g_union_info_get_field(info, C.gint(i))
		readFieldInfo(fi, &out.Fields[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
		// do discriminator values here too
		ci := C.g_union_info_get_discriminator(info, C.gint(i))
		readConstantInfo(ci, &out.DiscriminatorValues[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(ci)))
	}
	n = int(C.g_union_info_get_n_methods(info))
	out.Methods = make([]FunctionInfo, n)
	for i := 0; i < n; i++ {
		fi := C.g_union_info_get_method(info, C.gint(i))
		readFunctionInfo(fi, &out.Methods[i])
		C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(fi)))
	}
	out.Discriminated = fromgbool(C.g_union_info_is_discriminated(info))
	out.DiscriminatorOffset = int(C.g_union_info_get_discriminator_offset(info))
	ti := C.g_union_info_get_discriminator_type(info)
	readTypeInfo(ti, &out.DiscriminatorType)
	C.g_base_info_unref((*C.GIBaseInfo)(unsafe.Pointer(ti)))
	// discriminator values handled above
	out.Size = uintptr(C.g_union_info_get_size(info))
	out.Alignment = uintptr(C.g_union_info_get_alignment(info))
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

func readTypeInfo(info *C.GITypeInfo, out *TypeInfo) {
	readBaseInfo((*C.GIBaseInfo)(unsafe.Pointer(info)), &out.BaseInfo)
	out.IsPointer = fromgbool(C.g_type_info_is_pointer(info))
	out.Tag = TypeTag(C.g_type_info_get_tag(info))
	// TODO ParamTypes
	bi := C.g_type_info_get_interface(info)
	if bi != nil {
		readBaseInfo(bi, &out.Interface)
		C.g_base_info_unref(bi)
	}
	if out.Tag == TagArray {
		out.ArrayLength = int(C.g_type_info_get_array_length(info))
		out.ArrayFixedSize = int(C.g_type_info_get_array_fixed_size(info))
		out.IsZeroTerminated = fromgbool(C.g_type_info_is_zero_terminated(info))
		out.ArrayType = ArrayType(C.g_type_info_get_array_type(info))
	}
}

type Namespace struct {
	// TODO other fields
	Invalids		[]BaseInfo
	Functions		[]FunctionInfo
	Callbacks		[]CallableInfo
	Structs		[]StructInfo
	// TODO Boxed
	Enums		[]EnumInfo
	// TODO Flags
	Objects		[]ObjectInfo
	Interfaces		[]InterfaceInfo
	Constants		[]ConstantInfo
	Invalid0s		[]BaseInfo
	Unions		[]UnionInfo
	// TODO values
	Signals		[]SignalInfo
	VFuncs		[]VFuncInfo
	Properties		[]PropertyInfo
	Fields		[]FieldInfo
	Args			[]ArgInfo
	Types		[]TypeInfo
	Unresolveds	[]BaseInfo
}

func ReadNamespace(nsname string) (ns Namespace) {
	cns := (*C.gchar)(unsafe.Pointer(C.CString(nsname)))
	defer C.free(unsafe.Pointer(cns))
	if C.g_irepository_require(nil, cns, nil, 0, nil) == nil {
		panic("load failed")
	}
	n := int(C.g_irepository_get_n_infos(nil, cns))
	for i := 0; i < n; i++ {
		info := C.g_irepository_get_info(nil, cns, C.gint(i))
		switch InfoType(C.g_base_info_get_type(info)) {
		case TypeInvalid:
			var bi BaseInfo

			readBaseInfo(info, &bi)
			ns.Invalids = append(ns.Invalids, bi)
		case TypeFunction:
			var fi FunctionInfo

			readFunctionInfo((*C.GIFunctionInfo)(unsafe.Pointer(info)), &fi)
			ns.Functions = append(ns.Functions, fi)
		case TypeCallback:
			var ci CallableInfo

			// callbacks technically have type GICallableInfo but it looks like it's the same as GICallableInfo (and has no special methods of its own)
			readCallableInfo((*C.GICallableInfo)(unsafe.Pointer(info)), &ci)
			ns.Callbacks = append(ns.Callbacks, ci)
		case TypeStruct:
			var si StructInfo

			readStructInfo((*C.GIStructInfo)(unsafe.Pointer(info)), &si)
			ns.Structs = append(ns.Structs, si)
		case TypeBoxed:
			// TODO
		case TypeEnum:
			var ei EnumInfo

			readEnumInfo((*C.GIEnumInfo)(unsafe.Pointer(info)), &ei)
			ns.Enums = append(ns.Enums, ei)
		case TypeFlags:
			// TODO
		case TypeObject:
			var oi ObjectInfo

			readObjectInfo((*C.GIObjectInfo)(unsafe.Pointer(info)), &oi)
			ns.Objects = append(ns.Objects, oi)
		case TypeInterface:
			var ii InterfaceInfo

			readInterfaceInfo((*C.GIInterfaceInfo)(unsafe.Pointer(info)), &ii)
			ns.Interfaces = append(ns.Interfaces, ii)
		case TypeConstant:
			var ci ConstantInfo

			readConstantInfo((*C.GIConstantInfo)(unsafe.Pointer(info)), &ci)
			ns.Constants = append(ns.Constants, ci)
		case TypeInvalid0:
			var bi BaseInfo

			readBaseInfo(info, &bi)
			ns.Invalid0s = append(ns.Invalid0s, bi)
		case TypeUnion:
			var ui UnionInfo

			readUnionInfo((*C.GIUnionInfo)(unsafe.Pointer(info)), &ui)
			ns.Unions = append(ns.Unions, ui)
		case TypeValue:
			// TODO
		case TypeSignal:
			var si SignalInfo

			readSignalInfo((*C.GISignalInfo)(unsafe.Pointer(info)), &si)
			ns.Signals = append(ns.Signals, si)
		case TypeVFunc:
			var vi VFuncInfo

			readVFuncInfo((*C.GIVFuncInfo)(unsafe.Pointer(info)), &vi)
			ns.VFuncs = append(ns.VFuncs, vi)
		case TypeProperty:
			var pi PropertyInfo

			readPropertyInfo((*C.GIPropertyInfo)(unsafe.Pointer(info)), &pi)
			ns.Properties = append(ns.Properties, pi)
		case TypeField:
			var fi FieldInfo

			readFieldInfo((*C.GIFieldInfo)(unsafe.Pointer(info)), &fi)
			ns.Fields = append(ns.Fields, fi)
		case TypeArg:
			var ai ArgInfo

			readArgInfo((*C.GIArgInfo)(unsafe.Pointer(info)), &ai)
			ns.Args = append(ns.Args, ai)
		case TypeType:
			var ti TypeInfo

			readTypeInfo((*C.GITypeInfo)(unsafe.Pointer(info)), &ti)
			ns.Types = append(ns.Types, ti)
		case TypeUnresolved:
			var bi BaseInfo

			readBaseInfo(info, &bi)
			ns.Unresolveds = append(ns.Unresolveds, bi)
		default:
			panic("unknown info type")
		}
		C.g_base_info_unref(info)
	}
	return ns
}

type indenter struct {
	w	io.Writer
}
func (i *indenter) Write(p []byte) (n int, err error) {
	b := new(bytes.Buffer)
	err = json.Indent(b, p, "", "\t")
	if err != nil { return 0, err }
	return i.w.Write(b.Bytes())
}

func main() {
	e := json.NewEncoder(&indenter{os.Stdout})
	err := e.Encode(ReadNamespace("Gtk"))
	if err != nil { panic(err) }
}
