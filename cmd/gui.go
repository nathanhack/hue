package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nathanhack/hue/cmd/gui"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(guiCmd)
}

var guiCmd = &cobra.Command{
	Use:   "gui USERNAME GROUPNUM",
	Short: "Fullscreen GUI for regulating a group of lights",
	Long:  `Fullscreen GUI '`,
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		fmt.Println("gui called", strings.Join(args, ","))
		groupNum, err := strconv.Atoi(args[1])

		g := gui.GUI{
			UserName: args[0],
			GroupNum: groupNum,
		}

		err = g.Run()
		return
	},
}
