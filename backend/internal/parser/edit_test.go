package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenameBlock_Resource(t *testing.T) {
	dir := t.TempDir()

	mainTF := `resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(mainTF), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a second file that references the resource
	outputsTF := `output "instance_id" {
  value = aws_instance.web.id
}
`
	if err := os.WriteFile(filepath.Join(dir, "outputs.tf"), []byte(outputsTF), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := RenameBlock(RenameBlockRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_instance.web",
		NewName:       "app",
	})
	if err != nil {
		t.Fatalf("RenameBlock failed: %v", err)
	}

	// Verify the block was renamed
	if !strings.Contains(result.Content, `resource "aws_instance" "app"`) {
		t.Errorf("expected renamed block, got:\n%s", result.Content)
	}
	if strings.Contains(result.Content, `"web"`) {
		t.Errorf("expected old name to be gone, got:\n%s", result.Content)
	}

	// Verify attributes preserved
	if !strings.Contains(result.Content, `ami-12345`) {
		t.Errorf("expected attributes preserved, got:\n%s", result.Content)
	}

	// Verify references updated in other files
	outputsContent, err := os.ReadFile(filepath.Join(dir, "outputs.tf"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(outputsContent), "aws_instance.app.id") {
		t.Errorf("expected reference updated in outputs.tf, got:\n%s", string(outputsContent))
	}
	if strings.Contains(string(outputsContent), "aws_instance.web") {
		t.Errorf("expected old reference removed from outputs.tf, got:\n%s", string(outputsContent))
	}
}

func TestRenameBlock_Variable(t *testing.T) {
	dir := t.TempDir()

	content := `variable "region" {
  type    = string
  default = "us-east-1"
}
`
	if err := os.WriteFile(filepath.Join(dir, "variables.tf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := RenameBlock(RenameBlockRequest{
		WorkspacePath: dir,
		File:          "variables.tf",
		Address:       "var.region",
		NewName:       "aws_region",
	})
	if err != nil {
		t.Fatalf("RenameBlock failed: %v", err)
	}

	if !strings.Contains(result.Content, `variable "aws_region"`) {
		t.Errorf("expected renamed variable, got:\n%s", result.Content)
	}
}

func TestRenameBlock_NotFound(t *testing.T) {
	dir := t.TempDir()

	content := `resource "aws_instance" "web" {
  ami = "ami-12345"
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := RenameBlock(RenameBlockRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_instance.nonexistent",
		NewName:       "app",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent block")
	}
}

func TestAddNestedBlock(t *testing.T) {
	dir := t.TempDir()

	content := `resource "aws_security_group" "web" {
  name        = "web-sg"
  description = "Web security group"
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := AddNestedBlock(AddNestedBlockRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_security_group.web",
		BlockType:     "ingress",
		Attributes: map[string]string{
			"from_port":   "80",
			"to_port":     "80",
			"protocol":    `"tcp"`,
			"cidr_blocks": `["0.0.0.0/0"]`,
		},
	})
	if err != nil {
		t.Fatalf("AddNestedBlock failed: %v", err)
	}

	if !strings.Contains(result.Content, "ingress") {
		t.Errorf("expected ingress block, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "from_port") {
		t.Errorf("expected from_port attribute, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, `"tcp"`) {
		t.Errorf("expected protocol value, got:\n%s", result.Content)
	}
	// Verify parent attributes preserved
	if !strings.Contains(result.Content, `"web-sg"`) {
		t.Errorf("expected parent attributes preserved, got:\n%s", result.Content)
	}
}

func TestAddNestedBlock_NotFound(t *testing.T) {
	dir := t.TempDir()

	content := `resource "aws_instance" "web" {
  ami = "ami-12345"
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := AddNestedBlock(AddNestedBlockRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_security_group.missing",
		BlockType:     "ingress",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent block")
	}
}

func TestRemoveNestedBlock(t *testing.T) {
	dir := t.TempDir()

	content := `resource "aws_security_group" "web" {
  name = "web-sg"

  ingress {
    from_port = 80
    to_port   = 80
    protocol  = "tcp"
  }

  ingress {
    from_port = 443
    to_port   = 443
    protocol  = "tcp"
  }

  egress {
    from_port = 0
    to_port   = 0
    protocol  = "-1"
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Remove second ingress block (index 1)
	result, err := RemoveNestedBlock(RemoveNestedBlockRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_security_group.web",
		BlockType:     "ingress",
		Index:         1,
	})
	if err != nil {
		t.Fatalf("RemoveNestedBlock failed: %v", err)
	}

	// Port 80 ingress should remain
	if !strings.Contains(result.Content, "from_port = 80") {
		t.Errorf("expected first ingress preserved, got:\n%s", result.Content)
	}
	// Port 443 ingress should be gone
	if strings.Contains(result.Content, "443") {
		t.Errorf("expected second ingress removed, got:\n%s", result.Content)
	}
	// Egress should still be there
	if !strings.Contains(result.Content, "egress") {
		t.Errorf("expected egress preserved, got:\n%s", result.Content)
	}
}

func TestRemoveNestedBlock_IndexOutOfRange(t *testing.T) {
	dir := t.TempDir()

	content := `resource "aws_security_group" "web" {
  name = "web-sg"

  ingress {
    from_port = 80
    to_port   = 80
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := RemoveNestedBlock(RemoveNestedBlockRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_security_group.web",
		BlockType:     "ingress",
		Index:         5,
	})
	if err == nil {
		t.Fatal("expected error for out of range index")
	}
}

func TestRemoveAttribute(t *testing.T) {
	dir := t.TempDir()

	content := `resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"

  tags = {
    Name = "my-instance"
  }
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := RemoveAttribute(RemoveAttributeRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_instance.web",
		Attribute:     "tags",
	})
	if err != nil {
		t.Fatalf("RemoveAttribute failed: %v", err)
	}

	if strings.Contains(result.Content, "tags") {
		t.Errorf("expected tags removed, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "ami") {
		t.Errorf("expected other attributes preserved, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, "instance_type") {
		t.Errorf("expected instance_type preserved, got:\n%s", result.Content)
	}
}

func TestRemoveAttribute_NotFound(t *testing.T) {
	dir := t.TempDir()

	content := `resource "aws_instance" "web" {
  ami = "ami-12345"
}
`
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := RemoveAttribute(RemoveAttributeRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_instance.web",
		Attribute:     "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent attribute")
	}
}
