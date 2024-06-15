package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	loginPath string
)

// loginCmd represents the login command
var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to tinyr",
	Long:  `Log in to tinyr by visiting it in browser and pasting auth token to console.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Visit %v%v in your browser and copy/paste the token here.\n", url, loginPath)
		fmt.Scanln(&token)
		if token != "" {
			vip.Set("token", token)
			err := vip.WriteConfig()
			if err != nil {
				panic(err)
			}
			fmt.Println("ok")
		} else {
			fmt.Println("no token supplied 😔")
		}
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
	loginCmd.PersistentFlags().StringVar(&loginPath, "login_path", "/login", "Login path for tinyr")
}
