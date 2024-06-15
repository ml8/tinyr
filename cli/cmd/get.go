package cmd

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

func get(short string) {
	getUrl := fmt.Sprintf("%v/%v", url, short)
	req, err := http.NewRequest("GET", getUrl, nil)
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			fmt.Println(req.URL)
			return errors.New("")
		},
	}
	if err != nil {
		panic(err)
	}
	_, err = client.Do(req)
}

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a long URL associated with a short alias",
	Long: `Retrieves the long URL that is associated with a short alias.

tinyr get my-url`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("Short alias is required")
			return
		}
		short := args[0]
		get(short)
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
