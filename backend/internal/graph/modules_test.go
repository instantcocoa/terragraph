package graph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestExpandModulesLocalSource(t *testing.T) {
	// Create a temp workspace directory
	tmpDir := t.TempDir()

	// Create main.tf that references a local module
	mainTF := []byte(`
module "vpc" {
  source = "./modules/vpc"
  cidr   = "10.0.0.0/16"
}

resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t3.micro"
}
`)
	if err := os.WriteFile(filepath.Join(tmpDir, "main.tf"), mainTF, 0644); err != nil {
		t.Fatal(err)
	}

	// Create the module subdirectory with its own .tf files
	moduleDir := filepath.Join(tmpDir, "modules", "vpc")
	if err := os.MkdirAll(moduleDir, 0755); err != nil {
		t.Fatal(err)
	}

	moduleTF := []byte(`
variable "cidr" {
  description = "VPC CIDR block"
  type        = string
}

resource "aws_vpc" "main" {
  cidr_block = var.cidr
}

resource "aws_subnet" "public" {
  vpc_id     = aws_vpc.main.id
  cidr_block = "10.0.1.0/24"
}

output "vpc_id" {
  value = aws_vpc.main.id
}
`)
	if err := os.WriteFile(filepath.Join(moduleDir, "main.tf"), moduleTF, 0644); err != nil {
		t.Fatal(err)
	}

	// Parse the main workspace
	file, diags := hclsyntax.ParseConfig(mainTF, "main.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("parse error: %s", diags.Error())
	}

	body := file.Body.(*hclsyntax.Body)
	ext := NewExtractor()
	ext.ExtractFile("main.tf", body, mainTF)
	result := ext.Build()

	// Expand modules
	result.Nodes = ExpandModules(tmpDir, result.Nodes)

	// Find the module node
	var moduleNode *GraphNode
	for i, n := range result.Nodes {
		if n.Kind == KindModule && n.Name == "vpc" {
			moduleNode = &result.Nodes[i]
			break
		}
	}

	if moduleNode == nil {
		t.Fatal("expected to find module.vpc node")
	}

	if len(moduleNode.Children) == 0 {
		t.Fatal("expected module.vpc to have children")
	}

	// Module should have 4 children: variable, 2 resources, 1 output
	if len(moduleNode.Children) != 4 {
		t.Errorf("expected 4 children, got %d", len(moduleNode.Children))
		for _, c := range moduleNode.Children {
			t.Logf("  child: %s (%s)", c.Address, c.Kind)
		}
	}

	// Check that child addresses are prefixed with module.vpc
	for _, child := range moduleNode.Children {
		if child.Address[:len("module.vpc.")] != "module.vpc." {
			t.Errorf("expected child address to start with module.vpc., got %s", child.Address)
		}
	}

	// Verify specific child exists
	foundVPC := false
	foundSubnet := false
	for _, child := range moduleNode.Children {
		if child.Address == "module.vpc.aws_vpc.main" {
			foundVPC = true
		}
		if child.Address == "module.vpc.aws_subnet.public" {
			foundSubnet = true
		}
	}
	if !foundVPC {
		t.Error("expected to find child module.vpc.aws_vpc.main")
	}
	if !foundSubnet {
		t.Error("expected to find child module.vpc.aws_subnet.public")
	}
}

func TestExpandModulesSkipsRemoteSource(t *testing.T) {
	nodes := []GraphNode{
		{
			ID:           "module.consul",
			Kind:         KindModule,
			Name:         "consul",
			Address:      "module.consul",
			ModuleSource: "hashicorp/consul/aws",
		},
	}

	result := ExpandModules("/tmp/nonexistent", nodes)

	if len(result[0].Children) != 0 {
		t.Errorf("expected no children for remote module, got %d", len(result[0].Children))
	}
}

func TestExpandModulesSkipsGitSource(t *testing.T) {
	nodes := []GraphNode{
		{
			ID:           "module.example",
			Kind:         KindModule,
			Name:         "example",
			Address:      "module.example",
			ModuleSource: "git::https://example.com/module.git",
		},
	}

	result := ExpandModules("/tmp/nonexistent", nodes)

	if len(result[0].Children) != 0 {
		t.Errorf("expected no children for git module, got %d", len(result[0].Children))
	}
}

func TestIsLocalSource(t *testing.T) {
	tests := []struct {
		source string
		want   bool
	}{
		{"./modules/vpc", true},
		{"../shared/vpc", true},
		{"hashicorp/consul/aws", false},
		{"git::https://example.com/module.git", false},
		{"s3::https://bucket/module.zip", false},
		{"registry.terraform.io/hashicorp/consul/aws", false},
	}

	for _, tt := range tests {
		got := isLocalSource(tt.source)
		if got != tt.want {
			t.Errorf("isLocalSource(%q) = %v, want %v", tt.source, got, tt.want)
		}
	}
}
