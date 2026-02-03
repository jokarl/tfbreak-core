// Package plugin provides plugin discovery, loading, and execution for tfbreak.
//
// This file implements the plugin loader, which uses hashicorp/go-plugin to
// load and communicate with external plugin binaries over gRPC.
package plugin

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"

	sdkplugin "github.com/jokarl/tfbreak-plugin-sdk/plugin"
	"github.com/jokarl/tfbreak-plugin-sdk/tflint"
)

// LoadedPlugin represents a plugin that has been loaded and is ready for use.
type LoadedPlugin struct {
	// Info contains metadata about the discovered plugin.
	Info PluginInfo
	// RuleSet is the plugin's RuleSet implementation.
	RuleSet tflint.RuleSet
	// Client is the go-plugin client for managing the plugin process.
	Client *goplugin.Client
}

// Close terminates the plugin process.
func (p *LoadedPlugin) Close() {
	if p.Client != nil {
		p.Client.Kill()
	}
}

// Loader loads and manages plugin binaries.
type Loader struct {
	logger hclog.Logger
}

// NewLoader creates a new plugin loader.
func NewLoader() *Loader {
	return &Loader{
		logger: hclog.New(&hclog.LoggerOptions{
			Name:   "tfbreak-plugin-loader",
			Level:  hclog.Warn,
			Output: os.Stderr,
		}),
	}
}

// NewLoaderWithLogger creates a new plugin loader with a custom logger.
func NewLoaderWithLogger(logger hclog.Logger) *Loader {
	return &Loader{
		logger: logger,
	}
}

// Load loads a single plugin from the given path.
// The plugin process is started and a gRPC connection is established.
func (l *Loader) Load(info PluginInfo) (*LoadedPlugin, error) {
	// Verify the plugin binary exists and is executable
	if _, err := os.Stat(info.Path); err != nil {
		return nil, fmt.Errorf("plugin binary not found: %s", info.Path)
	}

	// Create the plugin client configuration
	clientConfig := &goplugin.ClientConfig{
		HandshakeConfig: sdkplugin.Handshake,
		Plugins:         sdkplugin.PluginMap,
		Cmd:             exec.Command(info.Path),
		Logger:          l.logger,
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolGRPC,
		},
	}

	// Create the client
	client := goplugin.NewClient(clientConfig)

	// Connect via RPC
	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to connect to plugin %s: %w", info.Name, err)
	}

	// Request the plugin
	raw, err := rpcClient.Dispense(sdkplugin.PluginName)
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("failed to dispense plugin %s: %w", info.Name, err)
	}

	// Assert the plugin type
	ruleSet, ok := raw.(tflint.RuleSet)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("plugin %s does not implement tflint.RuleSet", info.Name)
	}

	return &LoadedPlugin{
		Info:    info,
		RuleSet: ruleSet,
		Client:  client,
	}, nil
}

// LoadAll loads all plugins from the given list of plugin info.
// Returns a slice of loaded plugins and any errors encountered.
// Plugins that fail to load are skipped but errors are accumulated.
func (l *Loader) LoadAll(plugins []PluginInfo) ([]*LoadedPlugin, []error) {
	var loaded []*LoadedPlugin
	var errors []error

	for _, info := range plugins {
		if !info.Enabled {
			continue
		}

		plugin, err := l.Load(info)
		if err != nil {
			errors = append(errors, fmt.Errorf("plugin %s: %w", info.Name, err))
			continue
		}

		loaded = append(loaded, plugin)
	}

	return loaded, errors
}

// CloseAll closes all loaded plugins.
func CloseAll(plugins []*LoadedPlugin) {
	for _, p := range plugins {
		p.Close()
	}
}
