package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

type pbSchemaSpec struct {
	Message string `json:"message"`
	ProtoPaths []string `json:"proto_paths"`
	ImportPaths []string `json:"import_paths"`
}

type pbLoadedSchema struct {
	msgType protoreflect.MessageType
}

func pbLoadSchema(specArg any, stderr io.Writer) (*pbLoadedSchema, error) {
	specStr, ok := specArg.(string)
	if !ok {
		return nil, errors.New("spec should be string")
	}

	var spec pbSchemaSpec
	err := json.Unmarshal([]byte(specStr), &spec)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal spec: %w", err)
	}

	if spec.Message == "" {
		return nil, errors.New("need `message`")
	}

	if len(spec.ProtoPaths) == 0 {
		return nil, errors.New("need at least one `proto_paths`")
	}

	tempDir, err := os.MkdirTemp("", "pbjq-*")
	if err != nil {
		return nil, fmt.Errorf("failed to MkdirTemp: %w", err)
	}
	defer os.RemoveAll(tempDir)

	dsPath := filepath.Join(tempDir, "ds")
	protocArgs := []string{
		"-o", dsPath,
		"--experimental_allow_proto3_optional",
		"--include_imports",
	}

	for _, importPath := range spec.ImportPaths {
		protocArgs = append(protocArgs, "-I", importPath)
	}

	for _, protoPath := range spec.ProtoPaths {
		absProtoPath, err := filepath.Abs(protoPath)
		if err != nil {
			return nil, fmt.Errorf("bad proto path: %w", err)
		}
		protocArgs = append(protocArgs, absProtoPath)
	}

	cmd := exec.Command("protoc", protocArgs...)
	cmd.Stderr = stderr
	cmd.Stdout = stderr
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("protoc call failed: %w", err)
	}

	fdsData, err := os.ReadFile(dsPath)
	if err != nil {
		return nil, fmt.Errorf("failed reading descriptor set: %w", err)
	}

	fds := &descriptorpb.FileDescriptorSet{}
	err = proto.Unmarshal(fdsData, fds)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshalling descriptor set: %w", err)
	}

	files, err := protodesc.NewFiles(fds)
	if err != nil {
		return nil, fmt.Errorf("failed creating protodesc files: %w", err)
	}

	descriptor, err := files.FindDescriptorByName(protoreflect.FullName(spec.Message))
	if err != nil {
		return nil, fmt.Errorf("unable to find message descriptor by name: %w", err)
	}

	md, ok := descriptor.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, errors.New("`message` should be message name")
	}

	return &pbLoadedSchema{msgType: dynamicpb.NewMessageType(md)}, nil
}

func pbDecode(schemaArg any, inputValue any) (map[string]any, error) {
	schema, ok := schemaArg.(*pbLoadedSchema)
	if !ok {
		return nil, errors.New("schema arg has unexpected type")
	}

	inputStr, ok := inputValue.(string)
	if !ok {
		return nil, errors.New("input should be []byte")
	}

	msg := schema.msgType.New()
	err := proto.Unmarshal([]byte(inputStr), msg.Interface())
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshal input value: %w", err)
	}

	pj, err := protojson.Marshal(msg.Interface())
	if err != nil {
		return nil, fmt.Errorf("unable to dump protojson: %w", err)
	}

	var result map[string]any
	err = json.Unmarshal(pj, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal protojson: %w", err)
	}

	return result, nil
}
