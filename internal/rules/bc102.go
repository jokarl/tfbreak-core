package rules

import (
	"fmt"

	"github.com/jokarl/tfbreak-core/internal/types"
)

// BC102 detects invalid moved blocks
type BC102 struct{}

func init() {
	Register(&BC102{})
}

func (r *BC102) ID() string {
	return "BC102"
}

func (r *BC102) Name() string {
	return "invalid-moved-block"
}

func (r *BC102) Description() string {
	return "A moved block has invalid syntax or type mismatch between from/to addresses"
}

func (r *BC102) DefaultSeverity() types.Severity {
	return types.SeverityBreaking
}

func (r *BC102) Documentation() *RuleDoc {
	return &RuleDoc{
		ID:              r.ID(),
		Name:            r.Name(),
		DefaultSeverity: r.DefaultSeverity(),
		Description:     r.Description(),
		ExampleOld: `resource "aws_s3_bucket" "logs" {
  bucket = "my-logs"
}`,
		ExampleNew: `# Invalid: cannot move resource to module
moved {
  from = aws_s3_bucket.logs
  to   = module.storage  # Type mismatch!
}`,
		Remediation: `Fix the moved block to have matching types:
1. Resource to resource:
   moved {
     from = aws_s3_bucket.logs
     to   = aws_s3_bucket.application_logs
   }

2. Module to module:
   moved {
     from = module.old_vpc
     to   = module.network
   }

Moved blocks cannot change address types (resource ↔ module).`,
	}
}

func (r *BC102) Evaluate(old, new *types.ModuleSnapshot) []*types.Finding {
	var findings []*types.Finding

	for _, moved := range new.MovedBlocks {
		// Check for type mismatch (resource to module or vice versa)
		fromIsResource := types.IsResourceAddress(moved.From)
		fromIsModule := types.IsModuleAddress(moved.From)
		toIsResource := types.IsResourceAddress(moved.To)
		toIsModule := types.IsModuleAddress(moved.To)

		if fromIsResource && toIsModule {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Moved block has type mismatch: cannot move resource %q to module %q", moved.From, moved.To),
			).WithNewLocation(&moved.DeclRange)

			findings = append(findings, finding)
			continue
		}

		if fromIsModule && toIsResource {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Moved block has type mismatch: cannot move module %q to resource %q", moved.From, moved.To),
			).WithNewLocation(&moved.DeclRange)

			findings = append(findings, finding)
			continue
		}

		// If neither is a valid resource or module address, that's also invalid
		if !fromIsResource && !fromIsModule {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Moved block has invalid 'from' address: %q", moved.From),
			).WithNewLocation(&moved.DeclRange)

			findings = append(findings, finding)
			continue
		}

		if !toIsResource && !toIsModule {
			finding := types.NewFinding(
				r.ID(),
				r.Name(),
				r.DefaultSeverity(),
				fmt.Sprintf("Moved block has invalid 'to' address: %q", moved.To),
			).WithNewLocation(&moved.DeclRange)

			findings = append(findings, finding)
			continue
		}
	}

	return findings
}
