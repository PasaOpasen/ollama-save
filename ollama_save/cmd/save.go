/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ollama_save/util"
)

var (
	apath string
)

// saveCmd represents the save command
var saveCmd = &cobra.Command{
	Use:   "save <model1> <model2:tag> ...",
	Short: "Saves ollama model(s) to archive",
	Long: `Saves ollama model(s) to archive`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		err := util.ExportModels(ollama_path, args, apath)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(saveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// saveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:

	// var items []string
	// saveCmd.Flags().StringArrayVarP(&items, "model", "", []string{}, "model name with or without tag")
	saveCmd.Flags().StringVarP(&apath, "outpath", "o", "result.tar.gz", "path to result tar.gz with models")
}
