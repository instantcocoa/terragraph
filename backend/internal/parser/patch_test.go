package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPatchAttribute_Resource(t *testing.T) {
	dir := t.TempDir()
	tfFile := filepath.Join(dir, "main.tf")

	original := `resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t2.micro"

  tags = {
    Name = "my-instance"
  }
}
`
	if err := os.WriteFile(tfFile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := PatchAttribute(PatchRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_instance.web",
		Attribute:     "instance_type",
		Value:         `"t3.small"`,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result.Content, `"t3.small"`) {
		t.Errorf("expected patched content to contain t3.small, got:\n%s", result.Content)
	}

	// Verify other content is preserved
	if !strings.Contains(result.Content, `ami-12345`) {
		t.Errorf("expected ami to be preserved, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, `my-instance`) {
		t.Errorf("expected tags to be preserved, got:\n%s", result.Content)
	}

	// Verify file was actually written
	content, err := os.ReadFile(tfFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"t3.small"`) {
		t.Errorf("expected written file to contain t3.small")
	}
}

func TestPatchAttribute_Variable(t *testing.T) {
	dir := t.TempDir()
	tfFile := filepath.Join(dir, "variables.tf")

	original := `variable "region" {
  type    = string
  default = "us-east-1"
}

variable "count" {
  type    = number
  default = 1
}
`
	if err := os.WriteFile(tfFile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := PatchAttribute(PatchRequest{
		WorkspacePath: dir,
		File:          "variables.tf",
		Address:       "var.region",
		Attribute:     "default",
		Value:         `"eu-west-1"`,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result.Content, `"eu-west-1"`) {
		t.Errorf("expected patched content to contain eu-west-1, got:\n%s", result.Content)
	}

	// Other variable should be preserved
	if !strings.Contains(result.Content, `variable "count"`) {
		t.Errorf("expected other variable to be preserved, got:\n%s", result.Content)
	}
}

func TestPatchAttribute_Output(t *testing.T) {
	dir := t.TempDir()
	tfFile := filepath.Join(dir, "outputs.tf")

	original := `output "ip" {
  value       = aws_instance.web.public_ip
  description = "The public IP"
}
`
	if err := os.WriteFile(tfFile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := PatchAttribute(PatchRequest{
		WorkspacePath: dir,
		File:          "outputs.tf",
		Address:       "output.ip",
		Attribute:     "description",
		Value:         `"The public IP address"`,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result.Content, `"The public IP address"`) {
		t.Errorf("expected patched description, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, `aws_instance.web.public_ip`) {
		t.Errorf("expected value to be preserved, got:\n%s", result.Content)
	}
}

func TestPatchAttribute_Locals(t *testing.T) {
	dir := t.TempDir()
	tfFile := filepath.Join(dir, "locals.tf")

	original := `locals {
  env    = "staging"
  region = "us-east-1"
}
`
	if err := os.WriteFile(tfFile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := PatchAttribute(PatchRequest{
		WorkspacePath: dir,
		File:          "locals.tf",
		Address:       "local.env",
		Attribute:     "env",
		Value:         `"production"`,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result.Content, `"production"`) {
		t.Errorf("expected patched local, got:\n%s", result.Content)
	}
	if !strings.Contains(result.Content, `us-east-1`) {
		t.Errorf("expected other local to be preserved, got:\n%s", result.Content)
	}
}

func TestPatchAttribute_BoolValue(t *testing.T) {
	dir := t.TempDir()
	tfFile := filepath.Join(dir, "main.tf")

	original := `resource "aws_instance" "web" {
  ami                    = "ami-12345"
  associate_public_ip    = false
}
`
	if err := os.WriteFile(tfFile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := PatchAttribute(PatchRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_instance.web",
		Attribute:     "associate_public_ip",
		Value:         "true",
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result.Content, "true") {
		t.Errorf("expected bool value to be patched, got:\n%s", result.Content)
	}
}

func TestPatchAttribute_BlockNotFound(t *testing.T) {
	dir := t.TempDir()
	tfFile := filepath.Join(dir, "main.tf")

	original := `resource "aws_instance" "web" {
  ami = "ami-12345"
}
`
	if err := os.WriteFile(tfFile, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := PatchAttribute(PatchRequest{
		WorkspacePath: dir,
		File:          "main.tf",
		Address:       "aws_instance.nonexistent",
		Attribute:     "ami",
		Value:         `"ami-99999"`,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent block")
	}
}

func TestParseAddress(t *testing.T) {
	tests := []struct {
		addr      string
		wantType  string
		wantLabels []string
		wantErr   bool
	}{
		{"aws_instance.web", "resource", []string{"aws_instance", "web"}, false},
		{"var.region", "variable", []string{"region"}, false},
		{"local.env", "locals", nil, false},
		{"output.ip", "output", []string{"ip"}, false},
		{"data.aws_ami.latest", "data", []string{"aws_ami", "latest"}, false},
		{"module.vpc", "module", []string{"vpc"}, false},
		{"invalid", "", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			gotType, gotLabels, err := parseAddress(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAddress(%q) error = %v, wantErr %v", tt.addr, err, tt.wantErr)
				return
			}
			if gotType != tt.wantType {
				t.Errorf("parseAddress(%q) type = %q, want %q", tt.addr, gotType, tt.wantType)
			}
			if len(gotLabels) != len(tt.wantLabels) {
				t.Errorf("parseAddress(%q) labels = %v, want %v", tt.addr, gotLabels, tt.wantLabels)
			} else {
				for i := range gotLabels {
					if gotLabels[i] != tt.wantLabels[i] {
						t.Errorf("parseAddress(%q) label[%d] = %q, want %q", tt.addr, i, gotLabels[i], tt.wantLabels[i])
					}
				}
			}
		})
	}
}
