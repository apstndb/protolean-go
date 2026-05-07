// Package protolean converts protocol buffer messages to the LEAN format
// without an intermediate JSON representation.
package protolean

import (
	"encoding/base64"
	"fmt"

	"github.com/apstndb/protolean-go/lean"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// MarshalOptions configures how a proto.Message is converted to LEAN.
type MarshalOptions struct {
	// EmitDefaultValues causes fields set to their zero value to be emitted.
	EmitDefaultValues bool

	// EmitDefaultValuesForTypes emits default fields only for matching types.
	EmitDefaultValuesForTypes []protoreflect.FullName

	// EmitDefaultValuesForMessage is a predicate for per-type default emission.
	EmitDefaultValuesForMessage func(protoreflect.MessageDescriptor) bool
}

// Marshal converts a proto.Message to LEAN using default options.
func Marshal(m proto.Message) (string, error) {
	return MarshalOptions{}.Marshal(m)
}

// Marshal converts a proto.Message to LEAN.
func (o MarshalOptions) Marshal(m proto.Message) (string, error) {
	if m == nil {
		return "", nil
	}
	v, err := o.marshalMessage(m.ProtoReflect())
	if err != nil {
		return "", err
	}
	return lean.Encode(v)
}

func (o MarshalOptions) marshalMessage(pm protoreflect.Message) (any, error) {
	if v, ok, err := o.marshalWKT(pm); ok {
		if err != nil {
			return nil, err
		}
		return v, nil
	}

	emitDefaults := o.shouldEmitDefaults(pm.Descriptor())
	result := make(map[string]any)

	if emitDefaults {
		desc := pm.Descriptor()
		for i := 0; i < desc.Fields().Len(); i++ {
			fd := desc.Fields().Get(i)
			if fd.ContainingOneof() != nil && !pm.Has(fd) {
				continue
			}
			if (fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind) && !pm.Has(fd) {
				continue
			}
			v := pm.Get(fd)
			if fd.IsList() && v.List().Len() == 0 {
				continue
			}
			if fd.IsMap() && v.Map().Len() == 0 {
				continue
			}
			val, err := o.marshalFieldValue(fd, v)
			if err != nil {
				return nil, err
			}
			result[string(fd.Name())] = val
		}
	} else {
		var rangeErr error
		pm.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
			val, err := o.marshalFieldValue(fd, v)
			if err != nil {
				rangeErr = err
				return false
			}
			result[string(fd.Name())] = val
			return true
		})
		if rangeErr != nil {
			return nil, rangeErr
		}
	}

	return result, nil
}

func (o MarshalOptions) shouldEmitDefaults(md protoreflect.MessageDescriptor) bool {
	if o.EmitDefaultValues {
		return true
	}
	name := md.FullName()
	for _, n := range o.EmitDefaultValuesForTypes {
		if name == n {
			return true
		}
	}
	if o.EmitDefaultValuesForMessage != nil {
		return o.EmitDefaultValuesForMessage(md)
	}
	return false
}

func (o MarshalOptions) marshalFieldValue(fd protoreflect.FieldDescriptor, v protoreflect.Value) (any, error) {
	switch {
	case fd.IsList():
		return o.marshalList(fd, v.List())
	case fd.IsMap():
		return o.marshalMap(fd, v.Map())
	default:
		return o.marshalSingular(fd, v)
	}
}

func (o MarshalOptions) marshalSingular(fd protoreflect.FieldDescriptor, v protoreflect.Value) (any, error) {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return v.Bool(), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return v.Int(), nil
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return v.Int(), nil
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return v.Uint(), nil
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return v.Uint(), nil
	case protoreflect.FloatKind:
		return float32(v.Float()), nil
	case protoreflect.DoubleKind:
		return v.Float(), nil
	case protoreflect.StringKind:
		return v.String(), nil
	case protoreflect.BytesKind:
		return base64.StdEncoding.EncodeToString(v.Bytes()), nil
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return o.marshalMessage(v.Message())
	case protoreflect.EnumKind:
		return int32(v.Enum()), nil
	default:
		return nil, fmt.Errorf("protolean: unsupported field kind %v", fd.Kind())
	}
}

func (o MarshalOptions) marshalList(fd protoreflect.FieldDescriptor, list protoreflect.List) (any, error) {
	length := list.Len()
	result := make([]any, 0, length)
	for i := 0; i < length; i++ {
		v, err := o.marshalSingular(fd, list.Get(i))
		if err != nil {
			return nil, err
		}
		result = append(result, v)
	}
	return result, nil
}

func (o MarshalOptions) marshalMap(fd protoreflect.FieldDescriptor, m protoreflect.Map) (any, error) {
	result := make(map[string]any, m.Len())
	var rangeErr error
	m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		val, err := o.marshalSingular(fd.MapValue(), v)
		if err != nil {
			rangeErr = err
			return false
		}
		result[k.String()] = val
		return true
	})
	if rangeErr != nil {
		return nil, rangeErr
	}
	return result, nil
}

func (o MarshalOptions) marshalWKT(pm protoreflect.Message) (any, bool, error) {
	switch pm.Descriptor().FullName() {
	case "google.protobuf.Timestamp":
		return pm.Interface().(*timestamppb.Timestamp).AsTime().Format("2006-01-02T15:04:05Z"), true, nil
	case "google.protobuf.Duration":
		return pm.Interface().(*durationpb.Duration).AsDuration().String(), true, nil
	case "google.protobuf.Struct":
		m, err := o.marshalStruct(pm)
		return m, true, err
	case "google.protobuf.Value":
		v, err := o.marshalProtoValue(pm)
		return v, true, err
	case "google.protobuf.ListValue":
		lv, err := o.marshalProtoListValue(pm)
		return lv, true, err
	case "google.protobuf.FieldMask":
		return pm.Interface().(*fieldmaskpb.FieldMask).GetPaths(), true, nil
	case "google.protobuf.Any":
		v, err := o.marshalAny(pm)
		return v, true, err
	case "google.protobuf.BoolValue":
		return pm.Interface().(*wrapperspb.BoolValue).GetValue(), true, nil
	case "google.protobuf.Int32Value":
		return pm.Interface().(*wrapperspb.Int32Value).GetValue(), true, nil
	case "google.protobuf.Int64Value":
		return pm.Interface().(*wrapperspb.Int64Value).GetValue(), true, nil
	case "google.protobuf.UInt32Value":
		return pm.Interface().(*wrapperspb.UInt32Value).GetValue(), true, nil
	case "google.protobuf.UInt64Value":
		return pm.Interface().(*wrapperspb.UInt64Value).GetValue(), true, nil
	case "google.protobuf.FloatValue":
		return pm.Interface().(*wrapperspb.FloatValue).GetValue(), true, nil
	case "google.protobuf.DoubleValue":
		return pm.Interface().(*wrapperspb.DoubleValue).GetValue(), true, nil
	case "google.protobuf.StringValue":
		return pm.Interface().(*wrapperspb.StringValue).GetValue(), true, nil
	case "google.protobuf.BytesValue":
		return base64.StdEncoding.EncodeToString(pm.Interface().(*wrapperspb.BytesValue).GetValue()), true, nil
	case "google.protobuf.Empty":
		return map[string]any{}, true, nil
	}
	return nil, false, nil
}

func (o MarshalOptions) marshalStruct(pm protoreflect.Message) (map[string]any, error) {
	s := pm.Interface().(*structpb.Struct)
	result := make(map[string]any, len(s.Fields))
	for k, v := range s.Fields {
		val, err := o.marshalProtoValue(v.ProtoReflect())
		if err != nil {
			return nil, err
		}
		result[k] = val
	}
	return result, nil
}

func (o MarshalOptions) marshalProtoValue(pm protoreflect.Message) (any, error) {
	v := pm.Interface().(*structpb.Value)
	switch kind := v.Kind.(type) {
	case *structpb.Value_NullValue:
		return nil, nil
	case *structpb.Value_NumberValue:
		return kind.NumberValue, nil
	case *structpb.Value_StringValue:
		return kind.StringValue, nil
	case *structpb.Value_BoolValue:
		return kind.BoolValue, nil
	case *structpb.Value_StructValue:
		return o.marshalStruct(kind.StructValue.ProtoReflect())
	case *structpb.Value_ListValue:
		return o.marshalProtoListValue(kind.ListValue.ProtoReflect())
	default:
		return nil, fmt.Errorf("protolean: unsupported structpb.Value kind %T", kind)
	}
}

func (o MarshalOptions) marshalProtoListValue(pm protoreflect.Message) ([]any, error) {
	lv := pm.Interface().(*structpb.ListValue)
	result := make([]any, 0, len(lv.Values))
	for _, v := range lv.Values {
		val, err := o.marshalProtoValue(v.ProtoReflect())
		if err != nil {
			return nil, err
		}
		result = append(result, val)
	}
	return result, nil
}

func (o MarshalOptions) marshalAny(pm protoreflect.Message) (any, error) {
	a := pm.Interface().(*anypb.Any)
	mt, err := protoregistry.GlobalTypes.FindMessageByURL(a.GetTypeUrl())
	if err != nil {
		return map[string]any{
			"type_url": a.GetTypeUrl(),
			"value":    base64.StdEncoding.EncodeToString(a.GetValue()),
		}, nil
	}
	m := mt.New()
	if err := proto.Unmarshal(a.GetValue(), m.Interface()); err != nil {
		return map[string]any{
			"type_url": a.GetTypeUrl(),
			"value":    base64.StdEncoding.EncodeToString(a.GetValue()),
		}, nil
	}
	return o.marshalMessage(m)
}
