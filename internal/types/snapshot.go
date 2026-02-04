package types

// FileRange represents a location in a source file
type FileRange struct {
	Filename  string `json:"filename"`
	Line      int    `json:"line"`
	Column    int    `json:"column,omitempty"`
	EndLine   int    `json:"end_line,omitempty"`
	EndColumn int    `json:"end_column,omitempty"`
}

// ModuleSnapshot represents the extracted signature of a Terraform module
type ModuleSnapshot struct {
	// Path is the directory path of the module
	Path string `json:"path"`

	// Variables maps variable names to their signatures
	Variables map[string]*VariableSignature `json:"variables"`

	// Outputs maps output names to their signatures
	Outputs map[string]*OutputSignature `json:"outputs"`

	// Resources maps resource addresses (type.name) to their signatures
	Resources map[string]*ResourceSignature `json:"resources"`

	// Modules maps module call names to their signatures
	Modules map[string]*ModuleCallSignature `json:"modules"`

	// MovedBlocks contains all moved block declarations
	MovedBlocks []*MovedBlock `json:"moved_blocks"`

	// RequiredVersion is the terraform.required_version constraint
	RequiredVersion string `json:"required_version,omitempty"`

	// RequiredProviders maps provider names to their requirements
	RequiredProviders map[string]*ProviderRequirement `json:"required_providers,omitempty"`
}

// NewModuleSnapshot creates a new empty ModuleSnapshot
func NewModuleSnapshot(path string) *ModuleSnapshot {
	return &ModuleSnapshot{
		Path:              path,
		Variables:         make(map[string]*VariableSignature),
		Outputs:           make(map[string]*OutputSignature),
		Resources:         make(map[string]*ResourceSignature),
		Modules:           make(map[string]*ModuleCallSignature),
		MovedBlocks:       make([]*MovedBlock, 0),
		RequiredProviders: make(map[string]*ProviderRequirement),
	}
}

// VariableSignature represents the signature of a Terraform variable
type VariableSignature struct {
	// Name is the variable name
	Name string `json:"name"`

	// Type is the normalized type expression (e.g., "string", "list(string)")
	Type string `json:"type,omitempty"`

	// Default is the JSON-serialized default value, nil if no default
	Default interface{} `json:"default,omitempty"`

	// Description is the variable description
	Description string `json:"description,omitempty"`

	// Sensitive indicates if the variable is marked sensitive
	Sensitive bool `json:"sensitive,omitempty"`

	// Nullable indicates if the variable accepts null values.
	// nil means unspecified (defaults to true in Terraform 1.1+)
	// Pointer is used to distinguish unset from explicit false.
	Nullable *bool `json:"nullable,omitempty"`

	// Required is true if the variable has no default value
	Required bool `json:"required"`

	// ValidationCount is the number of validation blocks on this variable
	ValidationCount int `json:"validation_count,omitempty"`

	// Validations contains the validation blocks for this variable
	Validations []ValidationBlock `json:"validations,omitempty"`

	// DeclRange is the source location of the declaration
	DeclRange FileRange `json:"pos"`
}

// ValidationBlock represents a validation block on a variable
type ValidationBlock struct {
	// Condition is the raw condition expression as a string
	Condition string `json:"condition"`

	// ErrorMessage is the error message shown when validation fails
	ErrorMessage string `json:"error_message,omitempty"`
}

// HasDefault returns true if the variable has a default value
func (v *VariableSignature) HasDefault() bool {
	return !v.Required
}

// IsNullable returns the effective nullable value.
// Returns true if Nullable is nil (Terraform 1.1+ default) or explicitly true.
func (v *VariableSignature) IsNullable() bool {
	if v.Nullable == nil {
		return true // Terraform default since 1.1
	}
	return *v.Nullable
}

// OutputSignature represents the signature of a Terraform output
type OutputSignature struct {
	// Name is the output name
	Name string `json:"name"`

	// Description is the output description
	Description string `json:"description,omitempty"`

	// Sensitive indicates if the output is marked sensitive
	Sensitive bool `json:"sensitive,omitempty"`

	// DeclRange is the source location of the declaration
	DeclRange FileRange `json:"pos"`
}

// ResourceSignature represents the signature of a Terraform resource
type ResourceSignature struct {
	// Type is the resource type (e.g., "aws_s3_bucket")
	Type string `json:"type"`

	// Name is the resource name (e.g., "main")
	Name string `json:"name"`

	// Address is the full resource address (e.g., "aws_s3_bucket.main")
	Address string `json:"address"`

	// DeclRange is the source location of the declaration
	DeclRange FileRange `json:"pos"`
}

// ModuleCallSignature represents the signature of a Terraform module call
type ModuleCallSignature struct {
	// Name is the module call name
	Name string `json:"name"`

	// Source is the module source
	Source string `json:"source"`

	// Version is the module version constraint
	Version string `json:"version,omitempty"`

	// Address is the full module address (e.g., "module.vpc")
	Address string `json:"address"`

	// DeclRange is the source location of the declaration
	DeclRange FileRange `json:"pos"`
}

// MovedBlock represents a Terraform moved block
type MovedBlock struct {
	// From is the source address
	From string `json:"from"`

	// To is the destination address
	To string `json:"to"`

	// DeclRange is the source location of the declaration
	DeclRange FileRange `json:"pos"`
}

// ProviderRequirement represents a provider version requirement
type ProviderRequirement struct {
	// Source is the provider source (e.g., "hashicorp/aws")
	Source string `json:"source,omitempty"`

	// Version is the version constraint
	Version string `json:"version,omitempty"`
}

// IsResourceAddress returns true if the address refers to a resource (type.name format)
func IsResourceAddress(addr string) bool {
	// Resource addresses have the format "type.name" without "module." prefix
	if len(addr) == 0 {
		return false
	}
	if len(addr) >= 7 && addr[:7] == "module." {
		return false
	}
	// Check for at least one dot (type.name)
	for i := 0; i < len(addr); i++ {
		if addr[i] == '.' {
			return true
		}
	}
	return false
}

// IsModuleAddress returns true if the address refers to a module (module.name format)
func IsModuleAddress(addr string) bool {
	return len(addr) >= 7 && addr[:7] == "module."
}
