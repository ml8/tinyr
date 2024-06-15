package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
)

func add(short, long string) {
	fmt.Printf("%v -> %v\n", short, long)
	create := url + "/create"
	body := fmt.Sprintf("{ \"Short\": \"%v\", \"Long\": \"%v\" }", short, long)
	req, err := http.NewRequest("POST", create, strings.NewReader(body))
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

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a new short URL.",
	Long: `Create a new short URL given the short alias and the full URL

tinyr add my-url http://my-long-url.org/with/a/path`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) != 2 {
			fmt.Printf("Both short and long urls are required.\n")
			return
		}
		short := args[0]
		long := args[1]
		add(short, long)
	},
}

func init() {
	rootCmd.AddCommand(addCmd)
}
