package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/aqz236/port-fly/core/models"
	"github.com/aqz236/port-fly/core/utils"
)

var (
	// Build information
	version   = "dev"
	buildTime = "unknown"

	// Global flags
	cfgFile string
	verbose bool
	config  *models.Config
	logger  utils.Logger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "portfly",
	Short: "PortFly SSH Tunnel Manager",
	Long: `PortFly is a powerful SSH tunnel manager that provides:

• Local port forwarding (-L)
• Remote port forwarding (-R) 
• Dynamic port forwarding (SOCKS proxy)
• Web-based management interface
• Session persistence and management
• Connection pooling and monitoring

Examples:
  portfly start -L 8080:192.168.1.100:80 user@example.com
  portfly start -R 8080:localhost:3000 user@example.com
  portfly start -D 1080 user@example.com
  portfly list
  portfly stop session-123`,
	PersistentPreRunE: initializeConfig,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

// SetBuildInfo sets build information
func SetBuildInfo(v, bt string) {
	version = v
	buildTime = bt
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./configs/default.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose logging")

	// Version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("PortFly SSH Tunnel Manager\n")
			fmt.Printf("Version: %s\n", version)
			fmt.Printf("Built: %s\n", buildTime)
		},
	})
}

// initConfig reads in config file and ENV variables
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		// Search for config in working directory and config paths
		viper.AddConfigPath(".")
		viper.AddConfigPath("./configs")
		viper.AddConfigPath("$HOME/.portfly")
		viper.AddConfigPath("/etc/portfly")
		viper.SetConfigName("default")
		viper.SetConfigType("yaml")
	}

	// Environment variables
	viper.SetEnvPrefix("PORTFLY")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err != nil {
		// If config file not found, use defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
			os.Exit(1)
		}
	}
}

// initializeConfig initializes configuration and logger
func initializeConfig(cmd *cobra.Command, args []string) error {
	// Load configuration
	config = models.DefaultConfig()
	if err := viper.Unmarshal(config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Override log level if verbose flag is set
	if verbose {
		config.Logging.Level = "debug"
	}

	// Initialize logger
	loggerConfig := utils.LoggerConfig{
		Level:      config.Logging.Level,
		Format:     config.Logging.Format,
		Output:     config.Logging.Output,
		MaxSize:    config.Logging.MaxSize,
		MaxBackups: config.Logging.MaxBackups,
		MaxAge:     config.Logging.MaxAge,
		Compress:   config.Logging.Compress,
	}

	var err error
	logger, err = utils.NewLogger(loggerConfig)
	if err != nil {
		return fmt.Errorf("failed to initialize logger: %w", err)
	}

	return nil
}
