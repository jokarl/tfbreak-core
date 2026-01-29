package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-config-inspect/tfconfig"
	"github.com/jokarl/tfbreak-core/internal/pathfilter"
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

	// Parse nullable attributes (not supported by terraform-config-inspect)
	nullableMap, err := parseNullableAttributes(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse nullable attributes: %w", err)
	}

	// Parse validation blocks (not supported by terraform-config-inspect)
	validationMap, err := parseValidationBlocks(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to parse validation blocks: %w", err)
	}

	// Extract variables
	for name, v := range module.Variables {
		varSig := convertVariable(v)
		// Merge nullable attribute from direct HCL parsing
		if nullable, exists := nullableMap[name]; exists {
			varSig.Nullable = nullable
		}
		// Merge validation blocks from direct HCL parsing
		if validations, exists := validationMap[name]; exists {
			varSig.Validations = validations
			varSig.ValidationCount = len(validations)
		}
		snapshot.Variables[name] = varSig
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

// LoadWithFilter loads a Terraform module with path filtering applied.
// Note: terraform-config-inspect doesn't support per-file filtering,
// so this function uses the filter to determine which files' declarations
// should be included in the snapshot.
func LoadWithFilter(dir string, filter *pathfilter.Filter) (*types.ModuleSnapshot, error) {
	// Load the full module first
	snapshot, err := Load(dir)
	if err != nil {
		return nil, err
	}

	// If filter is nil, return unfiltered snapshot
	if filter == nil {
		return snapshot, nil
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	// Filter declarations based on their source file locations
	filteredVars := make(map[string]*types.VariableSignature)
	for name, v := range snapshot.Variables {
		if shouldIncludeFile(absDir, v.DeclRange.Filename, filter) {
			filteredVars[name] = v
		}
	}
	snapshot.Variables = filteredVars

	filteredOutputs := make(map[string]*types.OutputSignature)
	for name, o := range snapshot.Outputs {
		if shouldIncludeFile(absDir, o.DeclRange.Filename, filter) {
			filteredOutputs[name] = o
		}
	}
	snapshot.Outputs = filteredOutputs

	filteredResources := make(map[string]*types.ResourceSignature)
	for addr, r := range snapshot.Resources {
		if shouldIncludeFile(absDir, r.DeclRange.Filename, filter) {
			filteredResources[addr] = r
		}
	}
	snapshot.Resources = filteredResources

	filteredModules := make(map[string]*types.ModuleCallSignature)
	for name, m := range snapshot.Modules {
		if shouldIncludeFile(absDir, m.DeclRange.Filename, filter) {
			filteredModules[name] = m
		}
	}
	snapshot.Modules = filteredModules

	// Filter moved blocks
	var filteredMoved []*types.MovedBlock
	for _, moved := range snapshot.MovedBlocks {
		if shouldIncludeFile(absDir, moved.DeclRange.Filename, filter) {
			filteredMoved = append(filteredMoved, moved)
		}
	}
	snapshot.MovedBlocks = filteredMoved

	return snapshot, nil
}

// shouldIncludeFile checks if a file should be included based on the filter
func shouldIncludeFile(baseDir, filename string, filter *pathfilter.Filter) bool {
	// Get relative path from base directory
	relPath, err := filepath.Rel(baseDir, filename)
	if err != nil {
		return true // Include if we can't determine
	}

	// Normalize path separators for pattern matching
	relPath = filepath.ToSlash(relPath)

	match, err := filter.MatchFile(relPath)
	if err != nil {
		return true // Include if we can't match
	}

	return match
}
