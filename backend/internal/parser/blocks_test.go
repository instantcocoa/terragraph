package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddBlock_Resource(t *testing.T) {
	dir := t.TempDir()

	// Create an existing file with some content
	existing := `resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := AddBlock(AddBlockRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		BlockType:     "resource",
		ResourceType:  "aws_instance",
		Name:          "web",
		Attributes: map[string]string{
			"ami":           `"ami-12345"`,
			"instance_type": `"t2.micro"`,
		},
	})
	if err != nil {
		t.Fatalf("AddBlock failed: %v", err)
	}

	if !strings.Contains(result.Content, `resource "aws_instance" "web"`) {
		t.Errorf("expected resource block in output, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, `resource "aws_vpc" "main"`) {
		t.Errorf("expected original resource block preserved, got:\n%s", result.Content)
	}

	// Verify file was written
	content, err := os.ReadFile(filepath.Join(dir, "main.tf"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != result.Content {
		t.Error("file content does not match result content")
	}
}

func TestAddBlock_Variable(t *testing.T) {
	dir := t.TempDir()

	// Create empty file
	if err := os.WriteFile(filepath.Join(dir, "variables.tf"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := AddBlock(AddBlockRequest{
		WorkspacePath: dir,
		File:          "variables.tf",
		BlockType:     "variable",
		Name:          "region",
	})
	if err != nil {
		t.Fatalf("AddBlock failed: %v", err)
	}

	if !strings.Contains(result.Content, `variable "region"`) {
		t.Errorf("expected variable block, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "type") {
		t.Errorf("expected default type attribute, got:\n%s", result.Content)
	}
}

func TestAddBlock_Output(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "outputs.tf"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := AddBlock(AddBlockRequest{
		WorkspacePath: dir,
		File:          "outputs.tf",
		BlockType:     "output",
		Name:          "id",
		Attributes: map[string]string{
			"value": "aws_instance.web.id",
		},
	})
	if err != nil {
		t.Fatalf("AddBlock failed: %v", err)
	}

	if !strings.Contains(result.Content, `output "id"`) {
		t.Errorf("expected output block, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "aws_instance.web.id") {
		t.Errorf("expected value attribute, got:\n%s", result.Content)
	}
}

func TestAddBlock_NewFile(t *testing.T) {
	dir := t.TempDir()

	result, err := AddBlock(AddBlockRequest{
		WorkspacePath: dir,
		File:          "new.tf",
		BlockType:     "resource",
		ResourceType:  "aws_s3_bucket",
		Name:          "logs",
	})
	if err != nil {
		t.Fatalf("AddBlock failed: %v", err)
	}

	if !strings.Contains(result.Content, `resource "aws_s3_bucket" "logs"`) {
		t.Errorf("expected resource block in new file, got:\n%s", result.Content)
	}

	// Verify file was created
	if _, err := os.Stat(filepath.Join(dir, "new.tf")); os.IsNotExist(err) {
		t.Error("expected new.tf to be created")
	}
}

func TestRemoveBlock_Resource(t *testing.T) {
	dir := t.TempDir()

	content := `resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := RemoveBlock(RemoveBlockRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_instance.web",
	})
	if err != nil {
		t.Fatalf("RemoveBlock failed: %v", err)
	}

	if strings.Contains(result.Content, `resource "aws_instance" "web"`) {
		t.Errorf("expected resource block to be removed, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, `resource "aws_vpc" "main"`) {
		t.Errorf("expected other resource block preserved, got:\n%s", result.Content)
	}
}

func TestRemoveBlock_Variable(t *testing.T) {
	dir := t.TempDir()

	content := `variable "region" {
  type    = string
  default = "us-east-1"
}

variable "name" {
  type = string
}
`
	if err := os.WriteFile(filepath.Join(dir, "variables.tf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := RemoveBlock(RemoveBlockRequest{
		WorkspacePath: dir,
		File:          "variables.tf",
		Address:       "var.region",
	})
	if err != nil {
		t.Fatalf("RemoveBlock failed: %v", err)
	}

	if strings.Contains(result.Content, `variable "region"`) {
		t.Errorf("expected variable block to be removed, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, `variable "name"`) {
		t.Errorf("expected other variable block preserved, got:\n%s", result.Content)
	}
}

func TestRemoveBlock_NotFound(t *testing.T) {
	dir := t.TempDir()

	content := `resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := RemoveBlock(RemoveBlockRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_instance.missing",
	})
	if err == nil {
		t.Fatal("expected error for missing block")
	}
}

func TestAddBlock_InvalidType(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := AddBlock(AddBlockRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		BlockType:     "provider",
		Name:          "aws",
	})
	if err == nil {
		t.Fatal("expected error for unsupported block type")
	}
}
