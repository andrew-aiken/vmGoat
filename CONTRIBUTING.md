# Contribution Guidelines

This document outlines the standards and requirements for creating new scenarios and maintaining compatibility across the project.

## Creating a Scenario

Each scenario in vmGoat consists of infrastructure provisioning (Terraform) and configuration management (Ansible).
Follow these guidelines to ensure consistency and compatibility.

- Avoid creating cloud specific resources so multiple clouds can be used.
  - Don't incorporate access to cloud credentials (thats what [CloudGoats](https://github.com/RhinoSecurityLabs/cloudgoat/) for)

### Directory Structure

```
scenarios/
└── your-scenario-name/
    ├── README.md
    ├── terraform/
    │   ├── main.tf
    │   ├── settings.tf
    │   ├── data.tf
    │   └── outputs.tf
    └── ansible/
        ├── playbook.yaml
        ├── inventory.tmpl
        ├── flags.yaml
        ├── requirements.yaml (if needed)
        └── tasks/ (if needed)
            └── *.yaml
        └── templates/ (if needed)
            └── *
```

### README.md Requirements

Each scenario must include a README.md with the following format:

```markdown
# Scenario Name

Difficulty: X/10
Flags: Number of flags that should be found. (i.e. User & Root or 2)
Servers: X (Number of deployed VPS)

```bash
./vmGoat create your-scenario-name
\```

Brief description of the scenario and what the user will learn or accomplish.

---

Optional additional context or background information.
```

## Terraform Standards

### Required Files

1. **settings.tf** - Provider configuration and version constraints
2. **main.tf** - Primary resource definitions
3. **data.tf** - Data source definitions
4. **outputs.tf** - Output definitions
   1. This is where the entrypoint information and host addresses are returned

### settings.tf Requirements

```hcl
terraform {
  required_version = ">= 1.12.0"

  backend "local" {}

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    # Add other providers as needed
  }
}

provider "aws" {
  default_tags {
    tags = {
      scenario  = "your-scenario-name"
      source    = "vmGoat"
      terraform = "true"
      url       = "https://github.com/andrew-aiken/vmGoat"
    }
  }
}
```

### outputs.tf Requirements

The entrypoint will be displayed and can guide how to initially discover the deployed instances

At least one `host_*` is required, this is used when setting up hosts in the Ansible inventory.

```hcl
output "entrypoint" {
  value       = aws_instance.this.public_ip
  sensitive   = true
  description = "Entrypoint for the scenario"
}

output "host_main" {
  value       = aws_instance.this.public_ip
  sensitive   = true
  description = "The public IP address of the instance"
}
```

### Resource Standards

- **Instance Types**: Use cost-effective instance types (t3.micro, t3.small, t3.medium)
- **AMI**: Reference AMIs from data sources so scenarios are compatible across regions
- **Security Groups**: Reference shared security groups from data calls
  - All access to the deployed instances should be allowlisted
- **VPC/Subnets**: Reference shared VPC components from data calls
- **Key Pairs**: Use the shared "vmGoat" key pair for all provisioning

### data.tf Example

```hcl
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-*-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

data "aws_security_group" "this" {
  name = "vmGoat"
}

data "aws_subnets" "public" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.this.id]
  }

  filter {
    name   = "tag:Name"
    values = ["Public*"]
  }
}

data "aws_vpc" "this" {
  filter {
    name   = "tag:Name"
    values = ["vmGoat"]
  }
}
```

## Ansible Standards

### Playbook Structure

```yaml
---
- name: Setup [Scenario Name]
  hosts: main
  become: true
  vars:
    # Define scenario-specific variables
    variable_name: "value"
  tasks:
    - name: Import flags
      ansible.builtin.include_vars:
        file: flags.yaml

    # Your tasks here
    
    - name: Create root flag
      ansible.builtin.copy:
        dest: "/root/flag.txt"
        mode: '0400'
        owner: root
        group: root
        content: "{{ 'vmGoat{' + root_flag + '}' }}"
```

### inventory.tmpl Requirements

When the vmGoat program gets the Terraform outputs it replaces the `host_*` values in the inventory template.
This is to support deploying to multiple servers.

```ini
main ansible_ssh_host=host_main ansible_user=ubuntu
```

Keep the names of the host (i.e. `main`) the same as the variable name (i.e. `host_main`)

### flags.yaml Requirements

Location of where Ansible should source flags from, this can also be used to verify flags.

```yaml
---
user_flag: "user_flag_content_here"
root_flag: "root_flag_content_here"
# Add additional flags as needed
```

### Task Standards

- **Use fully qualified collection names** (e.g., `ansible.builtin.copy`)
- **Include task names** for all tasks
- **Use tags** to organize tasks by functionality
- **Use become: true** for privilege escalation when needed
- **Handle errors gracefully** with appropriate failed_when conditions
- **Idempotency**: The playbook should be able to be rerun without breaking or erroring

### Variable Naming

- Use lowercase with underscores: `my_variable_name`
- Prefix scenario-specific variables with scenario name when appropriate
- Use descriptive names that clearly indicate purpose

### requirements.yaml Format

If external collections are needed:

```yaml
---
collections:
  - name: community.general
    version: ">=1.0.0"
  - name: community.crypto
    version: ">=1.0.0"
```

## Formatting and Compatibility

### Code Formatting

- **Terraform**:
  - Use `terraform fmt` to format all `.tf` files
  - Use [tflint](https://github.com/terraform-linters/tflint) to check for unused Terraform resources
- **Ansible**: Use 2-space indentation, follow YAML best practices

### Testing Requirements

Before submitting:

1. Run `terraform validate` in the terraform directory
2. Run `ansible-playbook --syntax-check playbook.yaml` in the ansible directory
3. Test the complete scenario creation and destruction process

### Documentation

- **Comment on configurations, but don't state the obvious**
- **Use clear, descriptive resource and task names**

## Submission Guidelines

1. **Test thoroughly** before submitting
2. **Follow naming conventions** consistently
3. **Include all required files** in proper structure
4. **Document any special requirements** or dependencies
5. **Ensure compatibility** with existing vmGoat infrastructure

## Getting Help

If you need assistance or have questions about these guidelines, please:

1. Review existing scenarios for examples
3. Open an issue for clarification
4. Reach out to maintainers for complex scenarios

Thank you for helping make vmGoat better!
