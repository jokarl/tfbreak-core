# tfbreak Rules Reference

This document describes all rules implemented by tfbreak for detecting breaking and risky changes in Terraform modules.

## Severity Levels

- **BREAKING**: Changes that will definitely break existing consumers. These require immediate attention.
- **RISKY**: Changes that may break consumers depending on their usage patterns. Review recommended.
- **INFO**: Informational changes that are unlikely to cause issues but worth noting.

## Rule Categories

| Category | ID Range | Description |
|----------|----------|-------------|
| Variable Rules | BC001-BC005, RC003, RC006-RC008, RC012-RC013 | Changes to input variables |
| Output Rules | BC009-BC010, RC011 | Changes to output values |
| Resource/Module Rules | BC100-BC103, RC300-RC301 | Changes to resources and module calls |
| Version Rules | BC200-BC201 | Changes to version constraints |

## Rename Detection (Opt-in)

tfbreak can detect when variables or outputs are renamed rather than simply removed and added. This provides clearer feedback than separate "removed" and "added" findings.

**To enable rename detection**, add this to your `.tfbreak.hcl`:

```hcl
rename_detection {
  enabled              = true
  similarity_threshold = 0.85  # Default threshold (0.0-1.0)
}
```

When enabled, rename rules suppress the related removal/addition rules for matched pairs.

---

## Variable Rules

### BC001 - required-input-added

**Severity:** BREAKING

**Description:** A new variable was added without a default value, which will break existing callers.

**Trigger Condition:** A variable exists in the new version that did not exist in the old version, and the variable has no default value (making it required).

**Why it breaks:** Existing module consumers are not passing this variable. When they upgrade, Terraform will error with "The variable X is required but was not set."

**Example:**
```hcl
# NEW: This variable did not exist before
variable "api_key" {
  type        = string
  description = "API key for external service"
  # No default = required!
}
```

**Remediation:**
1. Add a default value to make the variable optional
2. Document the new required variable in your changelog
3. Use `# tfbreak:ignore required-input-added` if this is intentional

---

### BC002 - input-removed

**Severity:** BREAKING

**Description:** A variable was removed, which will break callers that provide this variable.

**Trigger Condition:** A variable exists in the old version but not in the new version.

**Why it breaks:** Existing consumers passing this variable will get an error: "An argument named X is not expected here."

**Example:**
```hcl
# OLD: This variable existed
variable "legacy_flag" {
  type    = bool
  default = false
}

# NEW: Variable is gone - callers still passing it will fail
```

**Remediation:**
1. Keep the variable but mark it as deprecated in the description
2. Use a validation block to warn users the variable is ignored
3. Use `# tfbreak:ignore input-removed` if removal is intentional

---

### BC003 - input-renamed (Opt-in)

**Severity:** BREAKING

**Description:** A required variable was renamed, which will break callers using the old name.

**Trigger Condition:** A required variable (no default) was removed, and a new required variable with a similar name was added. Requires rename detection to be enabled.

**Why it breaks:** Callers using the old variable name will get an error: "An argument named X is not expected here."

**Example:**
```hcl
# OLD
variable "api_key" {
  type        = string
  description = "API key for authentication"
}

# NEW - renamed to api_key_v2
variable "api_key_v2" {
  type        = string
  description = "API key for authentication (v2)"
}
```

**Remediation:**
1. Keep the old variable name for backward compatibility
2. Add the old variable as an alias that passes through to the new one
3. Coordinate with all callers to update to the new variable name
4. Use `# tfbreak:ignore input-renamed` if the rename is intentional and coordinated

**Note:** This rule only fires when rename detection is enabled. When it fires, it suppresses BC001 and BC002 for the matched variable pair.

---

### RC003 - input-renamed-optional (Opt-in)

**Severity:** RISKY

**Description:** An optional variable was renamed, which may break callers that explicitly set the old name.

**Trigger Condition:** An optional variable (has default) was removed, and a new optional variable with a similar name was added. Requires rename detection to be enabled.

**Why it's risky:** Callers who explicitly set the old variable will get an error. Callers who relied on the default are not affected.

**Example:**
```hcl
# OLD
variable "timeout" {
  type    = string
  default = "30s"
}

# NEW - renamed to timeout_ms
variable "timeout_ms" {
  type    = string
  default = "30000"
}
```

**Remediation:**
1. Keep the old variable name for backward compatibility
2. Add the old variable as a deprecated alias
3. Coordinate with callers who explicitly set this variable
4. Use `# tfbreak:ignore input-renamed-optional` if this is intentional

**Note:** This rule only fires when rename detection is enabled. When it fires, it suppresses BC002 for the matched variable.

---

### BC004 - input-type-changed

**Severity:** BREAKING

**Description:** A variable's type constraint changed, which may break callers passing values of the old type.

**Trigger Condition:** A variable exists in both versions but the `type` attribute changed.

**Why it breaks:** Callers passing values of the old type will get type mismatch errors.

**Example:**
```hcl
# OLD
variable "instance_count" {
  type = string  # Was accepting "5"
}

# NEW
variable "instance_count" {
  type = number  # Now requires 5 (unquoted)
}
```

**Remediation:**
1. Keep backward compatibility by accepting both types if possible
2. Document the type change in your changelog
3. Use `# tfbreak:ignore input-type-changed` if this is intentional

---

### BC005 - input-default-removed

**Severity:** BREAKING

**Description:** A variable's default value was removed, making it required.

**Trigger Condition:** A variable had a default value in the old version but doesn't have one in the new version.

**Why it breaks:** Callers relying on the default value must now explicitly provide a value.

**Example:**
```hcl
# OLD
variable "region" {
  type    = string
  default = "us-east-1"  # Callers could omit this
}

# NEW
variable "region" {
  type = string
  # No default - now required!
}
```

**Remediation:**
1. Keep the default value
2. Document that the variable is now required
3. Use `# tfbreak:ignore input-default-removed` if this is intentional

---

### RC006 - input-default-changed

**Severity:** RISKY

**Description:** A variable's default value changed, which may cause unexpected behavior.

**Trigger Condition:** A variable has a default value in both versions, but the values are different.

**Why it's risky:** Callers relying on the default may experience different behavior without realizing it.

**Example:**
```hcl
# OLD
variable "instance_type" {
  type    = string
  default = "t3.micro"
}

# NEW
variable "instance_type" {
  type    = string
  default = "t3.small"  # Larger instance, higher cost!
}
```

**Remediation:**
1. Document the change in your changelog
2. Consider whether callers expect this behavior change
3. Use `# tfbreak:ignore input-default-changed` if this is intentional

---

### RC007 - input-nullable-changed

**Severity:** RISKY

**Description:** A variable's nullable attribute changed, which may cause callers passing null to fail.

**Trigger Condition:** A variable's `nullable` attribute changed between versions.

**Why it's risky:** If nullable changed from true to false, callers passing `null` will get validation errors.

**Example:**
```hcl
# OLD
variable "optional_config" {
  type     = map(string)
  default  = {}
  nullable = true  # Accepts null
}

# NEW
variable "optional_config" {
  type     = map(string)
  default  = {}
  nullable = false  # No longer accepts null!
}
```

**Remediation:**
1. Document the nullability change
2. Ensure callers aren't passing null values
3. Use `# tfbreak:ignore input-nullable-changed` if this is intentional

---

### RC008 - input-sensitive-changed

**Severity:** RISKY

**Description:** A variable's sensitive attribute changed, which may affect downstream outputs and logging.

**Trigger Condition:** A variable's `sensitive` attribute changed between versions.

**Why it's risky:** Changing sensitivity affects how Terraform displays values in plans and outputs.

**Example:**
```hcl
# OLD
variable "database_password" {
  type      = string
  sensitive = false  # Was visible in plans
}

# NEW
variable "database_password" {
  type      = string
  sensitive = true  # Now hidden in plans
}
```

**Remediation:**
1. Document the sensitivity change
2. Consider impact on debugging and audit logs
3. Use `# tfbreak:ignore input-sensitive-changed` if this is intentional

---

### RC012 - validation-added

**Severity:** RISKY

**Description:** Validation blocks were added to a variable, which may cause deployment failures for consumers.

**Trigger Condition:** A variable's validation count increased (new validation blocks were added).

**Why it's risky:** Consumers passing values that don't meet the new validation criteria will experience deployment failures.

**Example:**
```hcl
# OLD
variable "environment" {
  type = string
}

# NEW
variable "environment" {
  type = string

  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "Environment must be dev, staging, or prod."
  }
}
```

**Remediation:**
1. Ensure the validation criteria aren't too restrictive
2. Document the new requirements in your changelog
3. Use `# tfbreak:ignore validation-added` if this is intentional

---

### RC013 - validation-value-removed

**Severity:** RISKY

**Description:** Allowed values were removed from a `contains()` validation, which may break consumers using those values.

**Trigger Condition:** A `contains([list], var.name)` validation pattern exists in both versions, and values were removed from the list.

**Why it's risky:** Consumers using the removed values will fail validation.

**Example:**
```hcl
# OLD
variable "environment" {
  validation {
    condition     = contains(["dev", "staging", "prod"], var.environment)
    error_message = "..."
  }
}

# NEW - "prod" removed!
variable "environment" {
  validation {
    condition     = contains(["dev", "staging"], var.environment)
    error_message = "..."
  }
}
```

**Remediation:**
1. Ensure no consumers are using the removed values
2. Document the deprecation of removed values
3. Use `# tfbreak:ignore validation-value-removed` if this is intentional

**Note:** This rule only detects changes to literal list values in `contains()` patterns. Dynamic lists like `contains(var.allowed, ...)` are not analyzed.

---

## Output Rules

### BC009 - output-removed

**Severity:** BREAKING

**Description:** An output was removed, which will break callers that reference this output.

**Trigger Condition:** An output exists in the old version but not in the new version.

**Why it breaks:** Consumers referencing this output (e.g., `module.foo.removed_output`) will get errors.

**Example:**
```hcl
# OLD
output "instance_ip" {
  value = aws_instance.main.public_ip
}

# NEW: Output is gone - consumers referencing it will fail
```

**Remediation:**
1. Keep the output but mark it deprecated
2. Return a placeholder value during deprecation period
3. Use `# tfbreak:ignore output-removed` if removal is intentional

---

### BC010 - output-renamed (Opt-in)

**Severity:** BREAKING

**Description:** An output was renamed, which will break callers referencing the old name.

**Trigger Condition:** An output was removed, and a new output with a similar name was added. Requires rename detection to be enabled.

**Why it breaks:** Consumers referencing the old output (e.g., `module.foo.old_output`) will get errors.

**Example:**
```hcl
# OLD
output "vpc_id" {
  value = aws_vpc.main.id
}

# NEW - renamed to main_vpc_id
output "main_vpc_id" {
  value = aws_vpc.main.id
}
```

**Remediation:**
1. Keep the old output name for backward compatibility
2. Add the old output as an alias pointing to the same value
3. Coordinate with all callers to update to the new output name
4. Use `# tfbreak:ignore output-renamed` if the rename is intentional and coordinated

**Note:** This rule only fires when rename detection is enabled. When it fires, it suppresses BC009 for the matched output.

---

### RC011 - output-sensitive-changed

**Severity:** RISKY

**Description:** An output's sensitive attribute changed, which affects plan visibility and downstream consumers.

**Trigger Condition:** An output's `sensitive` attribute changed between versions.

**Why it's risky:** Changing sensitivity affects how Terraform displays values and whether downstream modules can use the value in certain contexts.

**Example:**
```hcl
# OLD
output "connection_string" {
  value     = "..."
  sensitive = false
}

# NEW
output "connection_string" {
  value     = "..."
  sensitive = true  # Now hidden in plans
}
```

**Remediation:**
1. Document the sensitivity change
2. Consider impact on consumers who may be logging this output
3. Use `# tfbreak:ignore output-sensitive-changed` if this is intentional

---

## Resource and Module Rules

### BC100 - resource-removed-no-moved

**Severity:** BREAKING

**Description:** A resource was removed without a moved block, which will destroy the resource.

**Trigger Condition:** A resource exists in the old version but not in the new version, and there's no `moved` block redirecting it.

**Why it breaks:** Terraform will plan to destroy the resource, potentially causing data loss.

**Example:**
```hcl
# OLD
resource "aws_s3_bucket" "data" {
  bucket = "my-important-data"
}

# NEW: Resource is gone without moved block - will be destroyed!
```

**Remediation:**
1. Add a `moved` block if the resource was renamed:
   ```hcl
   moved {
     from = aws_s3_bucket.data
     to   = aws_s3_bucket.storage
   }
   ```
2. If intentionally removing, use `# tfbreak:ignore resource-removed-no-moved`

---

### BC101 - module-removed-no-moved

**Severity:** BREAKING

**Description:** A module was removed without a moved block, which will destroy the module's resources.

**Trigger Condition:** A module call exists in the old version but not in the new version, and there's no `moved` block redirecting it.

**Why it breaks:** All resources managed by the module will be destroyed.

**Example:**
```hcl
# OLD
module "vpc" {
  source = "./modules/vpc"
}

# NEW: Module is gone without moved block - all VPC resources will be destroyed!
```

**Remediation:**
1. Add a `moved` block if the module was renamed:
   ```hcl
   moved {
     from = module.vpc
     to   = module.network
   }
   ```
2. If intentionally removing, use `# tfbreak:ignore module-removed-no-moved`

---

### BC102 - invalid-moved-block

**Severity:** BREAKING

**Description:** A moved block has invalid syntax or type mismatch between from/to addresses.

**Trigger Condition:** A `moved` block exists but has invalid addresses (e.g., moving a resource to a module address).

**Why it breaks:** Terraform will error when parsing the configuration.

**Example:**
```hcl
# INVALID: Can't move a resource to a module
moved {
  from = aws_s3_bucket.old
  to   = module.storage  # Type mismatch!
}
```

**Remediation:**
1. Ensure both addresses are the same type (resource-to-resource or module-to-module)
2. Fix the address syntax

---

### BC103 - conflicting-moved

**Severity:** BREAKING

**Description:** Moved blocks have conflicts: duplicate from addresses, cycles, or non-existent to targets.

**Trigger Condition:** Multiple `moved` blocks have the same `from` address, form a cycle, or point to non-existent targets.

**Why it breaks:** Terraform cannot determine the correct state migration.

**Example:**
```hcl
# CONFLICT: Duplicate from address
moved {
  from = aws_s3_bucket.old
  to   = aws_s3_bucket.new1
}

moved {
  from = aws_s3_bucket.old  # Same from!
  to   = aws_s3_bucket.new2
}
```

**Remediation:**
1. Remove duplicate `moved` blocks
2. Break cycles in move chains
3. Ensure all `to` targets exist in the configuration

---

### RC300 - module-source-changed

**Severity:** RISKY

**Description:** A module call's source URL changed, which may point to a different module implementation.

**Trigger Condition:** A module call exists in both versions but the `source` attribute changed.

**Why it's risky:** The new source may point to a different module entirely, or a reorganized version with different behavior.

**Example:**
```hcl
# OLD
module "vpc" {
  source = "git::https://github.com/org/terraform-aws-vpc.git"
}

# NEW
module "vpc" {
  source = "registry.terraform.io/org/vpc/aws"  # Different source!
}
```

**Remediation:**
1. Verify the new source points to the same or compatible module
2. Test the change in a non-production environment
3. Use `# tfbreak:ignore module-source-changed` if this is intentional

---

### RC301 - module-version-changed

**Severity:** RISKY

**Description:** A module call's version constraint changed, which may pull in different module behavior.

**Trigger Condition:** A module call's `version` attribute changed between versions.

**Why it's risky:** Version changes may introduce breaking changes, new features, or bug fixes that affect behavior.

**Example:**
```hcl
# OLD
module "vpc" {
  source  = "registry.terraform.io/org/vpc/aws"
  version = "~> 3.0"
}

# NEW
module "vpc" {
  source  = "registry.terraform.io/org/vpc/aws"
  version = "~> 4.0"  # Major version bump!
}
```

**Remediation:**
1. Review the module's changelog for breaking changes
2. Test the version change in a non-production environment
3. Use `# tfbreak:ignore module-version-changed` if this is intentional

---

## Version Constraint Rules

### BC200 - terraform-version-constrained

**Severity:** BREAKING

**Description:** Terraform required_version constraint was added or changed, which may break CI pipelines using older versions.

**Trigger Condition:** The `required_version` in the `terraform` block changed.

**Why it breaks:** CI pipelines or developers using older Terraform versions will fail to init.

**Example:**
```hcl
# OLD
terraform {
  required_version = ">= 1.0"
}

# NEW
terraform {
  required_version = ">= 1.5"  # Excludes 1.0-1.4!
}
```

**Remediation:**
1. Ensure all consumers can upgrade to the required version
2. Document the version requirement change
3. Use `# tfbreak:ignore terraform-version-constrained` if this is intentional

---

### BC201 - provider-version-constrained

**Severity:** BREAKING

**Description:** Provider requirement was removed or changed, which may break consumers using different provider versions.

**Trigger Condition:** A provider requirement in `required_providers` was added, removed, or had its version constraint changed.

**Why it breaks:** Consumers may be using provider versions that no longer satisfy the constraint.

**Example:**
```hcl
# OLD
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 4.0"
    }
  }
}

# NEW
terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"  # Excludes 4.x!
    }
  }
}
```

**Remediation:**
1. Ensure all consumers can upgrade to the required provider version
2. Document the provider version requirement change
3. Use `# tfbreak:ignore provider-version-constrained` if this is intentional

---

## Suppressing Rules

You can suppress specific findings using inline annotations:

```hcl
# Suppress a single rule
# tfbreak:ignore required-input-added
variable "new_required_var" {
  type = string
}

# Suppress with a reason (recommended)
# tfbreak:ignore input-removed # deprecated in v2.0, removing in v3.0

# Suppress multiple rules
# tfbreak:ignore required-input-added,input-type-changed

# Suppress all rules for a block
# tfbreak:ignore all
```

See the [Annotations Guide](user-guide/annotations.md) for more details.
