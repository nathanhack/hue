package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

var ip string
var port int

var rootCmd = &cobra.Command{
	Use:   "hue",
	Short: "Does things with Philips Hue bulbs",
	Long:  `Does things with Philips Hue bulbs.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
