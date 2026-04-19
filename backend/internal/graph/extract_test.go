package graph

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestExtractSimpleWorkspace(t *testing.T) {
	src := []byte(`
variable "name" {
  description = "The name"
  type        = string
  default     = "hello"
}

locals {
  prefix = "test-${var.name}"
}

resource "aws_instance" "web" {
  ami           = "ami-12345"
  instance_type = "t3.micro"

  tags = {
    Name = local.prefix
  }
}

output "instance_id" {
  value = aws_instance.web.id
}
`)

	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("parse error: %s", diags.Error())
	}

	body := file.Body.(*hclsyntax.Body)
	ext := NewExtractor()
	ext.ExtractFile("test.tf", body, src)
	result := ext.Build()

	// Should have 4 nodes: variable, local, resource, output
	if len(result.Nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(result.Nodes))
	}

	// Check node kinds
	kinds := map[NodeKind]int{}
	for _, n := range result.Nodes {
		kinds[n.Kind]++
	}
	if kinds[KindVariable] != 1 {
		t.Errorf("expected 1 variable, got %d", kinds[KindVariable])
	}
	if kinds[KindLocal] != 1 {
		t.Errorf("expected 1 local, got %d", kinds[KindLocal])
	}
	if kinds[KindResource] != 1 {
		t.Errorf("expected 1 resource, got %d", kinds[KindResource])
	}
	if kinds[KindOutput] != 1 {
		t.Errorf("expected 1 output, got %d", kinds[KindOutput])
	}

	// Should have edges: local->var, resource->local, output->resource
	if len(result.Edges) < 2 {
		t.Errorf("expected at least 2 edges, got %d", len(result.Edges))
	}

	// Check resource address
	for _, n := range result.Nodes {
		if n.Kind == KindResource {
			if n.Address != "aws_instance.web" {
				t.Errorf("expected address aws_instance.web, got %s", n.Address)
			}
			if n.Provider != "aws" {
				t.Errorf("expected provider aws, got %s", n.Provider)
			}
		}
	}
}

func TestExtractMultipleResources(t *testing.T) {
	src := []byte(`
resource "aws_vpc" "main" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "public" {
  vpc_id     = aws_vpc.main.id
  cidr_block = "10.0.1.0/24"
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"]
}
`)

	file, diags := hclsyntax.ParseConfig(src, "test.tf", hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		t.Fatalf("parse error: %s", diags.Error())
	}

	body := file.Body.(*hclsyntax.Body)
	ext := NewExtractor()
	ext.ExtractFile("test.tf", body, src)
	result := ext.Build()

	if len(result.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(result.Nodes))
	}

	// Should have edge: subnet -> vpc
	foundEdge := false
	for _, e := range result.Edges {
		if e.Source == "resource.aws_subnet.public" && e.Target == "resource.aws_vpc.main" {
			foundEdge = true
		}
	}
	if !foundEdge {
		t.Errorf("expected edge from subnet to vpc, edges: %+v", result.Edges)
	}
}
