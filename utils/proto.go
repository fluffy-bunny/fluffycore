package utils

import (
	"encoding/json"
	"fmt"

	status "github.com/gogo/status"
	codes "google.golang.org/grpc/codes"
	protojson "google.golang.org/protobuf/encoding/protojson"
	proto "google.golang.org/protobuf/proto"
)

func ConvertStructToProto[FROM any](s FROM, pb proto.Message) error {
	// Marshal the struct to JSON.
	jsonBytes, err := json.Marshal(s)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to marshal struct to JSON: %v", err))
	}

	// Unmarshal the JSON to the protobuf message.
	if err := protojson.Unmarshal(jsonBytes, pb); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to unmarshal JSON to protobuf message: %v", err))
	}

	return nil
}

func ConvertProtoToStruct[TO any](pb proto.Message, s TO) error {
	// Marshal the protobuf message to JSON.
	jsonBytes, err := protojson.Marshal(pb)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to marshal protobuf message to JSON: %v", err))
	}

	// Unmarshal the JSON to the struct.
	if err := json.Unmarshal(jsonBytes, s); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to unmarshal JSON to struct: %v", err))
	}

	return nil
}

func ConvertProtoToMap(pb proto.Message) (map[string]interface{}, error) {
	// Marshal the protobuf message to JSON.
	jsonBytes, err := protojson.Marshal(pb)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to marshal protobuf message to JSON: %v", err))
	}

	// Unmarshal the JSON to a map.
	var m map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &m); err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to unmarshal JSON to map: %v", err))
	}

	return m, nil
}

func ConvertMapToProto(m map[string]interface{}, pb proto.Message) error {
	// Marshal the map to JSON.
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to marshal map to JSON: %v", err))
	}

	// Unmarshal the JSON to the protobuf message.
	if err := protojson.Unmarshal(jsonBytes, pb); err != nil {
		return status.Error(codes.Internal, fmt.Sprintf("failed to unmarshal JSON to protobuf message: %v", err))
	}

	return nil
}
