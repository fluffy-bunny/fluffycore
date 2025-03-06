package myplugin

type (
	IMyPlugin interface {
		DoSomething() (string, error)
	}
	MyPluginConfig struct {
		Enabled bool   `json:"enabled"`
		Name    string `json:"name"`
	}
	Config struct {
		MyPluginConfig MyPluginConfig `json:"myPluginConfig"`
	}
)
