package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func rm(short string) {
	fmt.Printf("deleting %v\n", short)
	rmUrl := url + "/delete"
	body := fmt.Sprintf("{ \"Short\": \"%v\" }", short)
	req, err := http.NewRequest("POST", rmUrl, strings.NewReader(body))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %v", token))
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("ok")
		resp.Body.Close()
	}
}

// rmCmd represents the rm command
var rmCmd = &cobra.Command{
	Use:   "rm",
	Short: "Remove a short URL",
	Long: `Remove a short URL given its short alias.

tinyr rm my-url`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 1 {
			fmt.Println("short url is required")
		}
		short := args[0]
		rm(short)
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
