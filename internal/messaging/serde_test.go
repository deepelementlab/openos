package messaging

import (
	"encoding/json"
	"testing"
)

func TestJSONSerializer_Serialize(t *testing.T) {
	s := NewJSONSerializer()
	data, err := s.Serialize(map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
	var result map[string]string
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["key"] != "value" {
		t.Errorf("expected value, got %s", result["key"])
	}
}

func TestJSONSerializer_Deserialize(t *testing.T) {
	s := NewJSONSerializer()
	input := `{"name":"test"}`
	var result map[string]string
	if err := s.Deserialize([]byte(input), &result); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if result["name"] != "test" {
		t.Errorf("expected test, got %s", result["name"])
	}
}

func TestJSONSerializer_Format(t *testing.T) {
	s := NewJSONSerializer()
	if s.Format() != FormatJSON {
		t.Errorf("expected FormatJSON, got %s", s.Format())
	}
}

func TestJSONSerializer_ContentType(t *testing.T) {
	s := NewJSONSerializer()
	if s.ContentType() != "application/json" {
		t.Errorf("expected application/json, got %s", s.ContentType())
	}
}

func TestJSONSerializer_RoundTrip(t *testing.T) {
	s := NewJSONSerializer()
	original := map[string]interface{}{"num": 42.0, "str": "hello", "bool": true}
	data, err := s.Serialize(original)
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	var result map[string]interface{}
	if err := s.Deserialize(data, &result); err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if result["num"] != 42.0 {
		t.Errorf("expected 42, got %v", result["num"])
	}
	if result["str"] != "hello" {
		t.Errorf("expected hello, got %v", result["str"])
	}
}

func TestProtobufSerializer_Serialize_NonProto(t *testing.T) {
	s := NewProtobufSerializer()
	_, err := s.Serialize("not a proto message")
	if err == nil {
		t.Error("expected error for non-proto message")
	}
}

func TestProtobufSerializer_Deserialize_NonProto(t *testing.T) {
	s := NewProtobufSerializer()
	err := s.Deserialize([]byte("data"), "not a proto message")
	if err == nil {
		t.Error("expected error for non-proto target")
	}
}

func TestProtobufSerializer_Format(t *testing.T) {
	s := NewProtobufSerializer()
	if s.Format() != FormatProtobuf {
		t.Errorf("expected FormatProtobuf, got %s", s.Format())
	}
}

func TestProtobufSerializer_ContentType(t *testing.T) {
	s := NewProtobufSerializer()
	if s.ContentType() != "application/x-protobuf" {
		t.Errorf("expected application/x-protobuf, got %s", s.ContentType())
	}
}

func TestSerializerFactory_GetSerializer_JSON(t *testing.T) {
	f := NewSerializerFactory()
	s, err := f.GetSerializer(FormatJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Format() != FormatJSON {
		t.Errorf("expected JSON format, got %s", s.Format())
	}
}

func TestSerializerFactory_GetSerializer_Protobuf(t *testing.T) {
	f := NewSerializerFactory()
	s, err := f.GetSerializer(FormatProtobuf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Format() != FormatProtobuf {
		t.Errorf("expected Protobuf format, got %s", s.Format())
	}
}

func TestSerializerFactory_GetSerializer_Unsupported(t *testing.T) {
	f := NewSerializerFactory()
	_, err := f.GetSerializer(FormatMsgPack)
	if err == nil {
		t.Error("expected error for unsupported format")
	}
}

func TestSerializerFactory_RegisterSerializer(t *testing.T) {
	f := NewSerializerFactory()
	custom := NewJSONSerializer()
	f.RegisterSerializer(FormatMsgPack, custom)
	s, err := f.GetSerializer(FormatMsgPack)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Error("expected serializer")
	}
}

func TestSerializerFactory_DefaultSerializer(t *testing.T) {
	f := NewSerializerFactory()
	s := f.DefaultSerializer()
	if s.Format() != FormatJSON {
		t.Errorf("expected default to be JSON, got %s", s.Format())
	}
}

func TestEventSerializer_SerializeEvent_JSON(t *testing.T) {
	es := NewEventSerializer()
	event, _ := NewEvent("test", map[string]string{"k": "v"})
	data, err := es.SerializeEvent(event, FormatJSON)
	if err != nil {
		t.Fatalf("serialize event failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
}

func TestEventSerializer_SerializeEvent_Protobuf_NotImplemented(t *testing.T) {
	es := NewEventSerializer()
	event, _ := NewEvent("test", nil)
	_, err := es.SerializeEvent(event, FormatProtobuf)
	if err == nil {
		t.Error("expected error for protobuf serialization")
	}
}

func TestEventSerializer_DeserializeEvent_JSON(t *testing.T) {
	es := NewEventSerializer()
	original, _ := NewEvent("test", map[string]string{"k": "v"})
	original.SetAgentID("agent-1")

	data, _ := es.SerializeEvent(original, FormatJSON)
	event, err := es.DeserializeEvent(data, FormatJSON)
	if err != nil {
		t.Fatalf("deserialize event failed: %v", err)
	}
	if event.Type != "test" {
		t.Errorf("expected type test, got %s", event.Type)
	}
	if event.AgentID != "agent-1" {
		t.Errorf("expected agent-1, got %s", event.AgentID)
	}
}

func TestEventSerializer_DeserializeEventWithPayload(t *testing.T) {
	es := NewEventSerializer()
	type payload struct {
		Name string `json:"name"`
	}
	original, _ := NewEvent("test", payload{Name: "hello"})

	data, _ := es.SerializeEvent(original, FormatJSON)
	var p payload
	event, err := es.DeserializeEventWithPayload(data, FormatJSON, &p)
	if err != nil {
		t.Fatalf("deserialize with payload failed: %v", err)
	}
	if p.Name != "hello" {
		t.Errorf("expected hello, got %s", p.Name)
	}
	if event.Type != "test" {
		t.Errorf("expected type test, got %s", event.Type)
	}
}

func TestEventSerializer_DeserializeEventWithPayload_NilPayloadObj(t *testing.T) {
	es := NewEventSerializer()
	original, _ := NewEvent("test", map[string]string{"k": "v"})
	data, _ := es.SerializeEvent(original, FormatJSON)
	event, err := es.DeserializeEventWithPayload(data, FormatJSON, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.Type != "test" {
		t.Errorf("expected type test, got %s", event.Type)
	}
}

func TestEventSerializer_DefaultFormat(t *testing.T) {
	es := NewEventSerializer()
	if es.DefaultFormat() != FormatJSON {
		t.Errorf("expected FormatJSON, got %s", es.DefaultFormat())
	}
}

func TestBackwardCompatibleSerialization_SerializeWithVersion(t *testing.T) {
	bc := NewBackwardCompatibleSerialization()
	event, _ := NewEvent("test", nil)
	event.SchemaVersion = "1.0"

	data, err := bc.SerializeWithVersion(event, "2.0")
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}
	if event.SchemaVersion != "1.0" {
		t.Errorf("expected original version to be preserved, got %s", event.SchemaVersion)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)
	if parsed["schema_version"] != "2.0" {
		t.Errorf("expected serialized version 2.0, got %v", parsed["schema_version"])
	}
}

func TestBackwardCompatibleSerialization_SerializeWithVersion_EmptyTarget(t *testing.T) {
	bc := NewBackwardCompatibleSerialization()
	event, _ := NewEvent("test", nil)
	event.SchemaVersion = "1.0"

	data, err := bc.SerializeWithVersion(event, "")
	if err != nil {
		t.Fatalf("serialize failed: %v", err)
	}

	var parsed map[string]interface{}
	json.Unmarshal(data, &parsed)
	if parsed["schema_version"] != "1.0" {
		t.Errorf("expected original version 1.0 when target empty, got %v", parsed["schema_version"])
	}
}

func TestBackwardCompatibleSerialization_DeserializeWithVersionMigration(t *testing.T) {
	bc := NewBackwardCompatibleSerialization()
	event, _ := NewEvent("test", nil)
	event.SchemaVersion = "1.0"

	data, _ := bc.SerializeWithVersion(event, "1.0")
	migrated, err := bc.DeserializeWithVersionMigration(data, "2.0")
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if migrated.SchemaVersion != "2.0" {
		t.Errorf("expected migrated version 2.0, got %s", migrated.SchemaVersion)
	}
}

func TestBackwardCompatibleSerialization_DeserializeWithVersionMigration_SameVersion(t *testing.T) {
	bc := NewBackwardCompatibleSerialization()
	event, _ := NewEvent("test", nil)
	event.SchemaVersion = "2.0"

	data, _ := bc.SerializeWithVersion(event, "2.0")
	migrated, err := bc.DeserializeWithVersionMigration(data, "2.0")
	if err != nil {
		t.Fatalf("deserialize failed: %v", err)
	}
	if migrated.SchemaVersion != "2.0" {
		t.Errorf("expected version to stay 2.0, got %s", migrated.SchemaVersion)
	}
}

func TestSerializationHelper_DetectFormat(t *testing.T) {
	h := &SerializationHelper{}
	if h.DetectFormat([]byte(`{"key":"value"}`)) != FormatJSON {
		t.Error("expected JSON format for object")
	}
	if h.DetectFormat([]byte(`[1,2,3]`)) != FormatJSON {
		t.Error("expected JSON format as default for array")
	}
	if h.DetectFormat(nil) != FormatJSON {
		t.Error("expected JSON as default for nil")
	}
}

func TestSerializationHelper_IsValidJSON(t *testing.T) {
	h := &SerializationHelper{}
	if !h.IsValidJSON([]byte(`{"key":"value"}`)) {
		t.Error("expected valid JSON")
	}
	if !h.IsValidJSON([]byte(`null`)) {
		t.Error("null is valid JSON")
	}
	if h.IsValidJSON([]byte(`{invalid`)) {
		t.Error("expected invalid JSON")
	}
}

func TestSerializationHelper_PrettyPrintJSON(t *testing.T) {
	h := &SerializationHelper{}
	input := `{"name":"test","value":42}`
	output, err := h.PrettyPrintJSON([]byte(input))
	if err != nil {
		t.Fatalf("pretty print failed: %v", err)
	}
	if len(output) <= len(input) {
		t.Error("expected pretty printed JSON to be larger")
	}
}

func TestSerializationHelper_PrettyPrintJSON_Invalid(t *testing.T) {
	h := &SerializationHelper{}
	_, err := h.PrettyPrintJSON([]byte(`invalid`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSerializationHelper_CompactJSON(t *testing.T) {
	h := &SerializationHelper{}
	input := `{"name" : "test" , "value" : 42}`
	output, err := h.CompactJSON([]byte(input))
	if err != nil {
		t.Fatalf("compact failed: %v", err)
	}
	if len(output) >= len(input) {
		t.Error("expected compacted JSON to be smaller or equal")
	}
}

func TestSerializationHelper_CompactJSON_Invalid(t *testing.T) {
	h := &SerializationHelper{}
	_, err := h.CompactJSON([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestSerializationFormatConstants(t *testing.T) {
	if FormatJSON != "json" {
		t.Errorf("expected json, got %s", FormatJSON)
	}
	if FormatProtobuf != "protobuf" {
		t.Errorf("expected protobuf, got %s", FormatProtobuf)
	}
	if FormatMsgPack != "msgpack" {
		t.Errorf("expected msgpack, got %s", FormatMsgPack)
	}
}
