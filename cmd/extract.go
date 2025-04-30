package cmd

import (
	"github.com/kream404/spoof/services/csv"
	"github.com/spf13/cobra"
)

var extractCmd = &cobra.Command{
	Use:  "extract",
	Long: `extract a new config file`,
	Run: func(cmd *cobra.Command, args []string) {
		ExtractConfigFile(args[0])
	},
}

func ExtractConfigFile(filepath string) error {
	records, _ := csv.ReadCSV(filepath)
	csv.MapFields(records)
	// println(fmt.Sprintln(fields))

	// for _, field := range fields {
	// 	println(field.Name, field.Type)
	// }
	return nil
}
