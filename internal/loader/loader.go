package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/jokarl/tfbreak-core/internal/types"
)

// Load loads a Terraform module from the given directory and returns its snapshot
func Load(dir string) (*types.ModuleSnapshot, error) {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("directory does not exist: %s", dir)
		}
		return nil, fmt.Errorf("failed to access directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", dir)
	}

	// Convert to absolute path
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Load using terraform-config-inspect
	module, diags := tfconfig.LoadModule(absDir)
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to load module: %s", diags.Error())
	}

	// Create snapshot
	snapshot := types.NewModuleSnapshot(absDir)

	// Extract variables
	for name, v := range module.Variables {
		snapshot.Variables[name] = convertVariable(v)
	}

	// Extract outputs
	for name, o := range module.Outputs {
		snapshot.Outputs[name] = convertOutput(o)
	}

	// Extract managed resources (not data sources)
	for addr, r := range module.ManagedResources {
		snapshot.Resources[addr] = convertResource(r)
	}

	// Extract module calls
	for name, m := range module.ModuleCalls {
		snapshot.Modules[name] = convertModuleCall(m)
	}

	// Extract required version
	if len(module.RequiredCore) > 0 {
		snapshot.RequiredVersion = module.RequiredCore[0]
	}

	// Extract required providers
	for name, p := range module.RequiredProviders {
		snapshot.RequiredProviders[name] = &types.ProviderRequirement{
			Source:  p.Source,
			Version: firstOrEmpty(p.VersionConstraints),
		}
	}

	// Parse moved blocks (not supported by terraform-config-inspect)
	movedBlocks, err := parseMovedBlocks(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse moved blocks: %w", err)
	}
	snapshot.MovedBlocks = movedBlocks

	return snapshot, nil
}

func convertVariable(v *tfconfig.Variable) *types.VariableSignature {
	return &types.VariableSignature{
		Name:        v.Name,
		Type:        v.Type,
		Default:     v.Default,
		Description: v.Description,
		Sensitive:   v.Sensitive,
		Required:    v.Required,
		DeclRange: types.FileRange{
			Filename: v.Pos.Filename,
			Line:     v.Pos.Line,
		},
	}
}

func convertOutput(o *tfconfig.Output) *types.OutputSignature {
	return &types.OutputSignature{
		Name:        o.Name,
		Description: o.Description,
		Sensitive:   o.Sensitive,
		DeclRange: types.FileRange{
			Filename: o.Pos.Filename,
			Line:     o.Pos.Line,
		},
	}
}

func convertResource(r *tfconfig.Resource) *types.ResourceSignature {
	return &types.ResourceSignature{
		Type:    r.Type,
		Name:    r.Name,
		Address: fmt.Sprintf("%s.%s", r.Type, r.Name),
		DeclRange: types.FileRange{
			Filename: r.Pos.Filename,
			Line:     r.Pos.Line,
		},
	}
}

func convertModuleCall(m *tfconfig.ModuleCall) *types.ModuleCallSignature {
	return &types.ModuleCallSignature{
		Name:    m.Name,
		Source:  m.Source,
		Version: m.Version,
		Address: fmt.Sprintf("module.%s", m.Name),
		DeclRange: types.FileRange{
			Filename: m.Pos.Filename,
			Line:     m.Pos.Line,
		},
	}
}

func firstOrEmpty(s []string) string {
	if len(s) > 0 {
		return s[0]
	}
	return ""
}
