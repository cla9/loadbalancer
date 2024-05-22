package main

import (
	"errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"lb/internal/agent"
	"os"
	"os/signal"
	"syscall"
)

type cli struct {
	cfg cfg
}

type cfg struct {
	agent.Config
}

func setupFlags(cmd *cobra.Command) error {
	cmd.Flags().String("config-file", "config/config.yaml", "Path to config file.")
	cmd.Flags().String("node-name", "test-id", "Unique server ID.")

	cmd.Flags().Int("grpc-port", 10000, "Port to bind xds server on.")
	cmd.Flags().Int("grpc-max-concurrent-streams", 1000000, "grpc max concurrent streams")
	cmd.Flags().Int("rest-port", 10001, "Port to bind rest api server on.")

	return viper.BindPFlags(cmd.Flags())
}

func main() {
	log.SetLevel(log.DebugLevel)

	cli := &cli{}
	cmd := &cobra.Command{
		Use:     "loadbalancer",
		PreRunE: cli.setupConfig,
		RunE:    cli.run,
	}
	if err := setupFlags(cmd); err != nil {
		log.Fatal(err)
	}
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func (c *cli) setupConfig(cmd *cobra.Command, args []string) error {
	var err error
	configFile, err := cmd.Flags().GetString("config-file")
	if err != nil {
		return err
	}
	viper.SetConfigFile(configFile)

	if err = viper.ReadInConfig(); err != nil {
		// it's ok if config file doesn't exist
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return err
		}
	}

	c.cfg.EnvoyConfig = viper.GetString("envoy-config")
	c.cfg.NodeName = viper.GetString("node-name")
	c.cfg.GrpcPort = viper.GetInt("grpc-port")
	c.cfg.GrpcMaxConcurrentStreams = viper.GetInt("grpc-max-concurrent-streams")
	c.cfg.RestPort = viper.GetInt("rest-port")

	return nil
}

func (c *cli) run(cmd *cobra.Command, args []string) error {
	var err error
	controlplane, err := agent.New(c.cfg.Config)
	if err != nil {
		return err
	}
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	<-sigc
	return controlplane.Shutdown()
}
