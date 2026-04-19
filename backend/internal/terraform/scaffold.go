package terraform

// Dependency describes a resource that should be created alongside another
type Dependency struct {
	BlockType    string            `json:"blockType"`    // "resource" or "data"
	ResourceType string            `json:"resourceType"` // e.g. "aws_subnet"
	Name         string            `json:"name"`         // suggested name
	Attributes   map[string]string `json:"attributes"`   // default HCL values
	LinkAttr     string            `json:"linkAttr"`     // attribute on the parent that references this
	LinkExpr     string            `json:"linkExpr"`     // HCL expression for the link
	Reason       string            `json:"reason"`       // why this dependency exists
	Required     bool              `json:"required"`     // whether the parent strictly needs this
}

// ScaffoldResult describes what to create when adding a resource
type ScaffoldResult struct {
	ResourceType string       `json:"resourceType"`
	Attributes   map[string]string `json:"attributes"`   // defaults for the main resource
	Dependencies []Dependency `json:"dependencies"`
}

// GetScaffold returns the scaffold template for a resource type
func GetScaffold(resourceType string) *ScaffoldResult {
	if scaffold, ok := scaffoldRegistry[resourceType]; ok {
		return &scaffold
	}
	// Return empty scaffold with no dependencies
	return &ScaffoldResult{
		ResourceType: resourceType,
		Attributes:   map[string]string{},
	}
}

// scaffoldRegistry maps resource types to their scaffold templates
var scaffoldRegistry = map[string]ScaffoldResult{
	// AWS
	"aws_instance": {
		ResourceType: "aws_instance",
		Attributes: map[string]string{
			"instance_type": `"t3.micro"`,
			"ami":           "data.aws_ami.latest.id",
			"subnet_id":     "aws_subnet.main.id",
			"tags": `{
    Name = "instance"
  }`,
		},
		Dependencies: []Dependency{
			{
				BlockType:    "data",
				ResourceType: "aws_ami",
				Name:         "latest",
				Attributes: map[string]string{
					"most_recent": "true",
					"owners":      `["amazon"]`,
				},
				LinkAttr: "ami",
				LinkExpr: "data.aws_ami.latest.id",
				Reason:   "AMI to launch the instance from",
				Required: true,
			},
			{
				BlockType:    "resource",
				ResourceType: "aws_vpc",
				Name:         "main",
				Attributes: map[string]string{
					"cidr_block":           `"10.0.0.0/16"`,
					"enable_dns_hostnames": "true",
					"enable_dns_support":   "true",
					"tags": `{
    Name = "main-vpc"
  }`,
				},
				LinkAttr: "",
				LinkExpr: "",
				Reason:   "VPC for networking",
				Required: true,
			},
			{
				BlockType:    "resource",
				ResourceType: "aws_subnet",
				Name:         "main",
				Attributes: map[string]string{
					"vpc_id":                  "aws_vpc.main.id",
					"cidr_block":              `"10.0.1.0/24"`,
					"map_public_ip_on_launch": "true",
					"tags": `{
    Name = "main-subnet"
  }`,
				},
				LinkAttr: "subnet_id",
				LinkExpr: "aws_subnet.main.id",
				Reason:   "Subnet to place the instance in",
				Required: true,
			},
			{
				BlockType:    "resource",
				ResourceType: "aws_security_group",
				Name:         "instance",
				Attributes: map[string]string{
					"vpc_id":      "aws_vpc.main.id",
					"name":        `"instance-sg"`,
					"description": `"Security group for instance"`,
				},
				LinkAttr: "vpc_security_group_ids",
				LinkExpr: "[aws_security_group.instance.id]",
				Reason:   "Security group for network access rules",
				Required: false,
			},
		},
	},

	"aws_s3_bucket": {
		ResourceType: "aws_s3_bucket",
		Attributes: map[string]string{
			"bucket": `"my-bucket-${random_id.suffix.hex}"`,
			"tags": `{
    Name = "my-bucket"
  }`,
		},
		Dependencies: []Dependency{
			{
				BlockType:    "resource",
				ResourceType: "random_id",
				Name:         "suffix",
				Attributes: map[string]string{
					"byte_length": "4",
				},
				LinkAttr: "bucket",
				LinkExpr: `"my-bucket-${random_id.suffix.hex}"`,
				Reason:   "Random suffix for globally unique bucket name",
				Required: false,
			},
		},
	},

	"aws_lambda_function": {
		ResourceType: "aws_lambda_function",
		Attributes: map[string]string{
			"function_name": `"my-function"`,
			"role":          "aws_iam_role.lambda.arn",
			"handler":       `"index.handler"`,
			"runtime":       `"nodejs20.x"`,
			"filename":      `"lambda.zip"`,
		},
		Dependencies: []Dependency{
			{
				BlockType:    "resource",
				ResourceType: "aws_iam_role",
				Name:         "lambda",
				Attributes: map[string]string{
					"name": `"lambda-role"`,
					"assume_role_policy": `jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })`,
				},
				LinkAttr: "role",
				LinkExpr: "aws_iam_role.lambda.arn",
				Reason:   "IAM role for Lambda execution",
				Required: true,
			},
		},
	},

	"aws_lb": {
		ResourceType: "aws_lb",
		Attributes: map[string]string{
			"name":               `"my-alb"`,
			"internal":           "false",
			"load_balancer_type": `"application"`,
			"subnets":            "aws_subnet.public[*].id",
			"security_groups":    "[aws_security_group.lb.id]",
		},
		Dependencies: []Dependency{
			{
				BlockType:    "resource",
				ResourceType: "aws_security_group",
				Name:         "lb",
				Attributes: map[string]string{
					"name":        `"lb-sg"`,
					"description": `"Security group for load balancer"`,
				},
				LinkAttr: "security_groups",
				LinkExpr: "[aws_security_group.lb.id]",
				Reason:   "Security group for the load balancer",
				Required: true,
			},
		},
	},

	"aws_rds_cluster": {
		ResourceType: "aws_rds_cluster",
		Attributes: map[string]string{
			"cluster_identifier": `"my-cluster"`,
			"engine":             `"aurora-mysql"`,
			"master_username":    `"admin"`,
			"master_password":    "var.db_password",
			"db_subnet_group_name": "aws_db_subnet_group.main.name",
		},
		Dependencies: []Dependency{
			{
				BlockType:    "resource",
				ResourceType: "aws_db_subnet_group",
				Name:         "main",
				Attributes: map[string]string{
					"name":       `"main"`,
					"subnet_ids": "aws_subnet.private[*].id",
				},
				LinkAttr: "db_subnet_group_name",
				LinkExpr: "aws_db_subnet_group.main.name",
				Reason:   "Subnet group for the database cluster",
				Required: true,
			},
			{
				BlockType:    "resource",
				ResourceType: "aws_security_group",
				Name:         "db",
				Attributes: map[string]string{
					"name":        `"db-sg"`,
					"description": `"Security group for database"`,
				},
				LinkAttr: "vpc_security_group_ids",
				LinkExpr: "[aws_security_group.db.id]",
				Reason:   "Security group for the database",
				Required: true,
			},
		},
	},

	// Hetzner
	"hcloud_server": {
		ResourceType: "hcloud_server",
		Attributes: map[string]string{
			"name":        `"my-server"`,
			"server_type": `"cx22"`,
			"image":       `"debian-12"`,
			"location":    `"nbg1"`,
			"ssh_keys":    "[hcloud_ssh_key.default.id]",
		},
		Dependencies: []Dependency{
			{
				BlockType:    "resource",
				ResourceType: "hcloud_ssh_key",
				Name:         "default",
				Attributes: map[string]string{
					"name":       `"default"`,
					"public_key": `file("~/.ssh/id_ed25519.pub")`,
				},
				LinkAttr: "ssh_keys",
				LinkExpr: "[hcloud_ssh_key.default.id]",
				Reason:   "SSH key for server access",
				Required: true,
			},
			{
				BlockType:    "resource",
				ResourceType: "hcloud_firewall",
				Name:         "default",
				Attributes: map[string]string{
					"name": `"default-fw"`,
				},
				LinkAttr: "firewall_ids",
				LinkExpr: "[hcloud_firewall.default.id]",
				Reason:   "Firewall for network security",
				Required: false,
			},
		},
	},

	// Google Cloud
	"google_compute_instance": {
		ResourceType: "google_compute_instance",
		Attributes: map[string]string{
			"name":         `"my-instance"`,
			"machine_type": `"e2-micro"`,
			"zone":         `"us-central1-a"`,
		},
		Dependencies: []Dependency{
			{
				BlockType:    "resource",
				ResourceType: "google_compute_network",
				Name:         "main",
				Attributes: map[string]string{
					"name":                    `"main-network"`,
					"auto_create_subnetworks": "true",
				},
				LinkAttr: "",
				LinkExpr: "",
				Reason:   "VPC network for the instance",
				Required: true,
			},
		},
	},
}

// GetAvailableScaffolds returns all resource types that have scaffold templates
func GetAvailableScaffolds() []string {
	var types []string
	for k := range scaffoldRegistry {
		types = append(types, k)
	}
	return types
}
