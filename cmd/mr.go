/*
Copyright (c) 2025 Tommy Hofmann
*/
package cmd

import (
	_"fmt"
	"github.com/thofma/bibi/lib/mr"
	"github.com/spf13/cobra"
)

// mrCmd represents the mr command
var mrCmd = &cobra.Command{
	Use:   "mr",
	Short: "Retrieve bibitems for mathematical literature using MRLookup",
	Long: `Query the free MR Lookup tool from the American Mathematical Society 
to retrieve bibliographic information about mathematical literature.

# Usage

    bibi mr author title year

A field can be omitted by supplying "-" and arguments in last position can be
left out.

# Examples

    bibi mr serre "a course in arithmetic"
    bibi mr serre "a course in arithmetic" 1973
    bibi mr serre - 1973`,
	Run: func(cmd *cobra.Command, args []string) {
		mr.Main(args)
	},
}

func init() {
	rootCmd.AddCommand(mrCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// mrCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// mrCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
