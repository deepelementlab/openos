package messaging

import (
	"encoding/json"
	"fmt"

	"google.golang.org/protobuf/proto"
)

// SerializationFormat defines the serialization format.
type SerializationFormat string

const (
	// FormatJSON uses JSON serialization.
	FormatJSON SerializationFormat = "json"
	// FormatProtobuf uses Protobuf serialization.
	FormatProtobuf SerializationFormat = "protobuf"
	// FormatMsgPack uses MessagePack serialization.
	FormatMsgPack SerializationFormat = "msgpack"
)

// Serializer provides serialization capabilities.
type Serializer interface {
	// Serialize serializes an object to bytes.
	Serialize(obj interface{}) ([]byte, error)

	// Deserialize deserializes bytes to an object.
	Deserialize(data []byte, obj interface{}) error

	// Format returns the serialization format.
	Format() SerializationFormat

	// ContentType returns the HTTP content type.
	ContentType() string
}

// JSONSerializer implements JSON serialization.
type JSONSerializer struct{}

// NewJSONSerializer creates a new JSON serializer.
func NewJSONSerializer() *JSONSerializer {
	return &JSONSerializer{}
}

// Serialize serializes an object to JSON.
func (s *JSONSerializer) Serialize(obj interface{}) ([]byte, error) {
	return json.Marshal(obj)
}

// Deserialize deserializes JSON to an object.
func (s *JSONSerializer) Deserialize(data []byte, obj interface{}) error {
	return json.Unmarshal(data, obj)
}

// Format returns the serialization format.
func (s *JSONSerializer) Format() SerializationFormat {
	return FormatJSON
}

// ContentType returns the HTTP content type.
func (s *JSONSerializer) ContentType() string {
	return "application/json"
}

// ProtobufSerializer implements Protobuf serialization.
type ProtobufSerializer struct{}

// NewProtobufSerializer creates a new Protobuf serializer.
func NewProtobufSerializer() *ProtobufSerializer {
	return &ProtobufSerializer{}
}

// Serialize serializes a Protobuf message to bytes.
func (s *ProtobufSerializer) Serialize(obj interface{}) ([]byte, error) {
	msg, ok := obj.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("object does not implement proto.Message")
	}
	return proto.Marshal(msg)
}

// Deserialize deserializes bytes to a Protobuf message.
func (s *ProtobufSerializer) Deserialize(data []byte, obj interface{}) error {
	msg, ok := obj.(proto.Message)
	if !ok {
		return fmt.Errorf("object does not implement proto.Message")
	}
	return proto.Unmarshal(data, msg)
}

// Format returns the serialization format.
func (s *ProtobufSerializer) Format() SerializationFormat {
	return FormatProtobuf
}

// ContentType returns the HTTP content type.
func (s *ProtobufSerializer) ContentType() string {
	return "application/x-protobuf"
}

// SerializerFactory creates serializers based on format.
type SerializerFactory struct {
	serializers map[SerializationFormat]Serializer
}

// NewSerializerFactory creates a new serializer factory.
func NewSerializerFactory() *SerializerFactory {
	return &SerializerFactory{
		serializers: map[SerializationFormat]Serializer{
			FormatJSON:     NewJSONSerializer(),
			FormatProtobuf: NewProtobufSerializer(),
		},
	}
}

// GetSerializer returns a serializer for the given format.
func (f *SerializerFactory) GetSerializer(format SerializationFormat) (Serializer, error) {
	serializer, exists := f.serializers[format]
	if !exists {
		return nil, fmt.Errorf("unsupported serialization format: %s", format)
	}
	return serializer, nil
}

// RegisterSerializer registers a custom serializer.
func (f *SerializerFactory) RegisterSerializer(format SerializationFormat, serializer Serializer) {
	f.serializers[format] = serializer
}

// DefaultSerializer returns the default JSON serializer.
func (f *SerializerFactory) DefaultSerializer() Serializer {
	return f.serializers[FormatJSON]
}

// EventSerializer provides event-specific serialization.
type EventSerializer struct {
	factory *SerializerFactory
}

// NewEventSerializer creates a new event serializer.
func NewEventSerializer() *EventSerializer {
	return &EventSerializer{
		factory: NewSerializerFactory(),
	}
}

// SerializeEvent serializes an event using the specified format.
func (s *EventSerializer) SerializeEvent(event *Event, format SerializationFormat) ([]byte, error) {
	serializer, err := s.factory.GetSerializer(format)
	if err != nil {
		return nil, err
	}

	// For backward compatibility, we wrap the event for JSON
	if format == FormatJSON {
		return serializer.Serialize(event)
	}

	// For Protobuf, we need to convert to proto.Event (would need proto definitions)
	return nil, fmt.Errorf("protobuf event serialization not yet implemented")
}

// DeserializeEvent deserializes an event using the specified format.
func (s *EventSerializer) DeserializeEvent(data []byte, format SerializationFormat) (*Event, error) {
	serializer, err := s.factory.GetSerializer(format)
	if err != nil {
		return nil, err
	}

	event := &Event{}
	if err := serializer.Deserialize(data, event); err != nil {
		return nil, fmt.Errorf("failed to deserialize event: %w", err)
	}

	return event, nil
}

// DeserializeEventWithPayload deserializes an event and its payload.
func (s *EventSerializer) DeserializeEventWithPayload(data []byte, format SerializationFormat, payloadObj interface{}) (*Event, error) {
	event, err := s.DeserializeEvent(data, format)
	if err != nil {
		return nil, err
	}

	// Deserialize payload if requested
	if payloadObj != nil && len(event.Payload) > 0 {
		payloadSerializer, _ := s.factory.GetSerializer(format)
		if err := payloadSerializer.Deserialize(event.Payload, payloadObj); err != nil {
			return nil, fmt.Errorf("failed to deserialize payload: %w", err)
		}
	}

	return event, nil
}

// DefaultFormat returns the default serialization format.
func (s *EventSerializer) DefaultFormat() SerializationFormat {
	return FormatJSON
}

// BackwardCompatibleSerialization handles backward-compatible serialization.
// It ensures that events can be read by consumers using older schema versions.
type BackwardCompatibleSerialization struct {
	serializer *EventSerializer
}

// NewBackwardCompatibleSerialization creates a new backward-compatible serializer.
func NewBackwardCompatibleSerialization() *BackwardCompatibleSerialization {
	return &BackwardCompatibleSerialization{
		serializer: NewEventSerializer(),
	}
}

// SerializeWithVersion serializes an event with schema version information.
func (s *BackwardCompatibleSerialization) SerializeWithVersion(event *Event, targetVersion string) ([]byte, error) {
	// Store the original schema version
	originalVersion := event.SchemaVersion

	// Temporarily set target version for serialization
	if targetVersion != "" {
		event.SchemaVersion = targetVersion
	}

	// Serialize
	data, err := s.serializer.SerializeEvent(event, FormatJSON)

	// Restore original version
	event.SchemaVersion = originalVersion

	return data, err
}

// DeserializeWithVersionMigration deserializes and migrates to current version.
func (s *BackwardCompatibleSerialization) DeserializeWithVersionMigration(data []byte, currentVersion string) (*Event, error) {
	event, err := s.serializer.DeserializeEvent(data, FormatJSON)
	if err != nil {
		return nil, err
	}

	// Check if migration is needed
	if event.SchemaVersion != currentVersion {
		// In a real implementation, this would apply version migrations
		// For now, we just update the schema version
		event.SchemaVersion = currentVersion
	}

	return event, nil
}

// SerializationHelper provides utility functions for serialization.
type SerializationHelper struct{}

// DetectFormat attempts to detect the serialization format from data.
func (h *SerializationHelper) DetectFormat(data []byte) SerializationFormat {
	// Simple heuristic: if it starts with '{', it's likely JSON
	if len(data) > 0 && data[0] == '{' {
		return FormatJSON
	}
	// Default to JSON
	return FormatJSON
}

// IsValidJSON checks if data is valid JSON.
func (h *SerializationHelper) IsValidJSON(data []byte) bool {
	var v interface{}
	return json.Unmarshal(data, &v) == nil
}

// PrettyPrintJSON returns a pretty-printed JSON string.
func (h *SerializationHelper) PrettyPrintJSON(data []byte) (string, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return "", err
	}

	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}

	return string(pretty), nil
}

// CompactJSON returns compact JSON without whitespace.
func (h *SerializationHelper) CompactJSON(data []byte) ([]byte, error) {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, err
	}

	return json.Marshal(v)
}
