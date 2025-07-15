package configuration

import (
	"embed"
	"os"
	"strings"

	mConfig "github.com/mitchs-dev/library-go/configuration"
	"github.com/mitchs-dev/library-go/loggingFormatter"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type InstanceMapConfig struct {
	Config     InstanceConfig
	Namespaces namespaceMap
}

// instanceMap is a map of instance IDs to InstanceConfig
type instanceMap map[int]InstanceMapConfig

// namespaceMap is a map of namespace names to NamespaceConfig
type namespaceMap map[string]NamespaceConfig

// Instances is a map of instance IDs to InstanceConfig
var Instances instanceMap

// Configuration struct for the application
type Configuration struct {
	Server Server `yaml:"server" json:"server"`
	// Cluster configuration
	Cluster Cluster `yaml:"cluster" json:"cluster"`
	// Redis configuration
	Redis Redis `yaml:"redis" json:"redis"`
	// Instance configurations
	Instances []InstanceConfig `yaml:"instances" json:"instances"`
}

// Server struct for the application
type Server struct {
	// The port on which the server will listen
	Port int `yaml:"port" json:"port"`
	// The host on which the server will listen
	Host string `yaml:"host" json:"host"`
	// The log format (json or text)
	LogFormat string `yaml:"log_format" json:"log_format"`
	// Timezone the server
	Timezone string `yaml:"timezone" json:"timezone"`
}

// Cluster struct for the application
type Cluster struct {
	// Enabled cluster mode (Cluster mode does not work yet)
	Enabled bool `yaml:"enabled" json:"enabled"`
	// Path to store the nodes cluster ID
	IDPath string `yaml:"id_path" json:"id_path"`
	// Cluster management configuration
	ClusterManagement ClusterManagement `yaml:"management" json:"management"`
}

// ClusterManagement struct for the application
type ClusterManagement struct {
	// The management database
	DB int `yaml:"db" json:"db"`
}

// Redis struct for the Badger database configuration
type Redis struct {
	// The port on which the Redis server is listening
	Port int `yaml:"port" json:"port"`
	// The host of the Redis server
	Host string `yaml:"host" json:"host"`
	// The password for the Redis server (If applicable)
	Password string `yaml:"password" json:"password"`
	// The maximum number of database instances
	MaxDBInstances int `yaml:"max_db_instances" json:"max_db_instances"`
}

// InstanceConfig struct for the application
type InstanceConfig struct {
	// The ID of the instance
	ID int `yaml:"id" json:"id"`
	// Namespaces for the application
	Namespaces []NamespaceConfig `yaml:"namespaces" json:"namespaces"`
}

// NamespaceConfig struct for the application
type NamespaceConfig struct {
	// The name of the namespace
	Name string `yaml:"name" json:"name"`
	// Options for this namespace
	NamespaceOptions `yaml:"options" json:"options"`
}

// NamespaceOptions struct for the application
type NamespaceOptions struct {
	// Timeout for the next client to accept the lock after it has been unlocked
	NextInQueueTimeout string `yaml:"next_in_queue_timeout" json:"next_in_queue_timeout"`
}

//go:embed default.json
var defaultConfigEmbed embed.FS

var (
	defaultConfigFilePath = "default.json" // Path name of the default configuration file
	configFileData        []byte           // The default configuration file data
	Context               *Configuration   // The configuration context
	logLevelEnvVar        = "LOG_LEVEL"    // The environment variable for the log level
	configFileEnvVar      = "CONFIG_FILE"  // The environment variable for the configuration file
	useConfigFilePath     string           // The configuration file to use
)

func init() {

	Context = &Configuration{}

	// Set the logging format
	log.SetFormatter(&loggingFormatter.JSONFormatter{
		Prefix:   "dislo-",
		Timezone: "UTC",
	})

	logLevelEnvVarValue := os.Getenv(logLevelEnvVar)
	if logLevelEnvVarValue == "" {
		log.SetLevel(log.InfoLevel)
	} else {
		switch strings.ToLower(logLevelEnvVarValue) {
		case "debug":
			log.SetLevel(log.DebugLevel)
		case "info":
			log.SetLevel(log.InfoLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		case "fatal":
			log.SetLevel(log.FatalLevel)
		case "panic":
			log.SetLevel(log.PanicLevel)
		default:
			log.Fatal("Invalid log level: ", logLevelEnvVarValue)
		}
	}

	// Use the stdout for logging
	log.SetOutput(os.Stdout)

	// Read the default configuration file
	defaultConfigData, err := defaultConfigEmbed.ReadFile(defaultConfigFilePath)
	if err != nil {
		log.Fatalf("Failed to read default configuration file: %v", err)
	}

	if useConfigFilePath = os.Getenv(configFileEnvVar); useConfigFilePath != "" {
		log.Debugf("Using configuration file: %s", useConfigFilePath)
		configFileData, err = os.ReadFile(useConfigFilePath)
		if err != nil {
			log.Fatalf("Failed to read configuration file: %v", err)
		}
	} else {
		log.Debugf("Using default configuration file: %s", defaultConfigFilePath)
		configFileData = defaultConfigData
	}

	var defaultConfigStruct map[interface{}]interface{}
	var configStruct map[interface{}]interface{}

	// Unmarshal the default configuration file
	if err := yaml.Unmarshal(defaultConfigData, &defaultConfigStruct); err != nil {
		log.Fatalf("Failed to unmarshal default configuration file: %v", err)
	}

	// Try to unmarshal the configuration file (YAML package is able to unmarshal JSON and YAML!)
	err = yaml.Unmarshal(configFileData, &configStruct)
	if err != nil {
		log.Errorf("Failed to unmarshal configuration file: %v", err)
	} else {
		log.Debug("Configuration file successfully unmarshalled as YAML")
	}

	// Merge the default configuration with the configuration file
	mergedStructs := mConfig.MergeWithDefault(defaultConfigStruct, configStruct)

	// Marshal the merged configuration struct
	mergedConfigData, err := yaml.Marshal(mergedStructs)
	if err != nil {
		log.Fatalf("Failed to marshal merged configuration struct: %v", err)
	}

	// Unmarshal the merged configuration struct
	err = yaml.Unmarshal(mergedConfigData, Context)
	if err != nil {
		log.Fatalf("Failed to unmarshal merged configuration struct: %v", err)
	}

	// Set the log format
	switch strings.ToLower(Context.Server.LogFormat) {
	case "json":
		log.SetFormatter(&loggingFormatter.JSONFormatter{
			Prefix:   "dislo-",
			Timezone: Context.Server.Timezone,
		})
	case "text":
		log.SetFormatter(&loggingFormatter.Formatter{})
	default:
		log.Fatalf("Invalid log format: %s - Must be 'json' or 'text'", Context.Server.LogFormat)
	}

	if len(Context.Instances) == 0 {
		log.Fatal("No instances found in configuration file")
	}
	log.Debugf("Mapping instances to IDs")
	Instances = make(instanceMap)
	for _, instance := range Context.Instances {
		if len(instance.Namespaces) == 0 {
			log.Fatalf("No namespaces found for instance %d", instance.ID)
		}
		instanceNamespaceMap := make(namespaceMap)
		for _, namespace := range instance.Namespaces {
			instanceNamespaceMap[namespace.Name] = namespace
		}
		instanceMapConfig := InstanceMapConfig{
			Config:     instance,
			Namespaces: instanceNamespaceMap,
		}
		Instances[instance.ID] = instanceMapConfig
	}
	log.Debugf("Instances: %v", Instances)

}
