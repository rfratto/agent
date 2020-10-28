package plugin

// What's the API we want to use here?
// - Serve
// - AddPlugin

// The plugin system should be responsible for launching and running the plugins. Maybe?
// DialPlugin should give

// Launch creates a gRPC connection to an executable on disk. stdin and stdout
// will be used to communicate with the plugin.

// Launch launches a new PluginClient from disk. The plugin should be an
// executable found in the system PATH or a full path to an executable.
//
// Communication to and from the client will occur over stdin/stdout.
func Launch(plugin string) (PluginClient, error) {
	panic("nyi")
}
