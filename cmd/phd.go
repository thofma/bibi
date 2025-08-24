/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"github.com/thofma/bibi/lib/phd"
	"github.com/spf13/cobra"
)

// phdCmd represents the phd command
var phdCmd = &cobra.Command{
	Use:   "phd",
	Short: "Retrieve bibitems for PhD theses in mathematics",
	Long: `Query the Mathematics Genealogy Project (https://www.genealogy.math.ndsu.nodak.edu/)
to retrieve bibliographic information about PhD theses in mathematics.

Only the (partial) name is supplied.

# Examples

bibi phd gauss 

bibi phd carl gauss`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("phd called")
		phd.Main(args)
	},
}

func init() {
	rootCmd.AddCommand(phdCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// phdCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// phdCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
