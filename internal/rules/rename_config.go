package rules

// RenameDetectionSettings holds the configuration for rename detection rules
type RenameDetectionSettings struct {
	Enabled             bool
	SimilarityThreshold float64
}

// DefaultRenameDetectionSettings returns the default settings (disabled)
func DefaultRenameDetectionSettings() *RenameDetectionSettings {
	return &RenameDetectionSettings{
		Enabled:             false,
		SimilarityThreshold: 0.85,
	}
}

// renameSettings is the current rename detection configuration
// This is set by the engine based on the loaded config
var renameSettings = DefaultRenameDetectionSettings()

// SetRenameDetectionSettings updates the rename detection configuration
func SetRenameDetectionSettings(settings *RenameDetectionSettings) {
	if settings == nil {
		renameSettings = DefaultRenameDetectionSettings()
		return
	}
	renameSettings = settings
}

// GetRenameDetectionSettings returns the current rename detection configuration
func GetRenameDetectionSettings() *RenameDetectionSettings {
	return renameSettings
}

// IsRenameDetectionEnabled returns whether rename detection is enabled
func IsRenameDetectionEnabled() bool {
	return renameSettings != nil && renameSettings.Enabled
}

// GetSimilarityThreshold returns the current similarity threshold
func GetSimilarityThreshold() float64 {
	if renameSettings == nil {
		return 0.85
	}
	return renameSettings.SimilarityThreshold
}
