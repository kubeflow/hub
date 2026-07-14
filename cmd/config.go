package cmd

type Config struct {
	DbFile             string   `mapstructure:"db-file" yaml:"db-file"`
	Hostname           string   `mapstructure:"hostname" yaml:"hostname"`
	Port               int      `mapstructure:"port" yaml:"port"`
	LibraryDirs        []string `mapstructure:"metadata-library-dir" yaml:"metadata-library-dir"`
	CORSAllowedOrigins []string `mapstructure:"cors-allowed-origins" yaml:"cors-allowed-origins"`
}

var cfg = Config{DbFile: "metadata.sqlite.db", Hostname: "localhost", Port: 8080, LibraryDirs: []string(nil)}

const EnvPrefix = "MR"
