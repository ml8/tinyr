package cmd

import (
	"fmt"
	"os"
	"path"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	url   string
	token string
	vip   *viper.Viper
)

const (
	configType     = "yaml"
	configFileName = ".tinyr.yaml"
)

func initConfig(cmd *cobra.Command) error {
	vip = viper.New()
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	vip.SetConfigFile(path.Join(home, configFileName))
	err = vip.ReadInConfig()
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil
		}
		return err
	}
	if token == "" && vip.IsSet("token") {
		token = fmt.Sprintf("%v", vip.Get("token"))
	}
	return nil
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "tinyr",
	Short: "Short url manipulation",
	Long:  `Command line interface for manipulating short urls stored in tinyr.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig(cmd)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&url, "url", "https://tinyr.us", "URL for tinyr")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Auth token for tinyr")
}
