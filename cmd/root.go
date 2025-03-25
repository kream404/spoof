package cmd

import (
	"fmt"
	"os"

	"github.com/kream404/scratch/fakers"
	"github.com/kream404/scratch/models"
	"github.com/kream404/scratch/services/json"

	"github.com/spf13/cobra"
)

var config_path string
var config models.FileConfig
var scaffold string
var scaffold_name string

var rootCmd = &cobra.Command{
	Use:   "scratch", // This is the name of your CLI tool
	Short: "A brief description of your CLI",
	Run: func(cmd *cobra.Command, args []string) {

		// multiple returns, _ indicates ignore the error. neato
		version, _ := cmd.Flags().GetBool("version")
		verbose, _ := cmd.Flags().GetBool("verbose")
		scaffold, _ := cmd.Flags().GetBool("scaffold")


		if version {
			versionCmd.Run(cmd, args)
			return
		}

		config, _ := json.LoadConfig(config_path)

		if(verbose && config != nil) {
			fmt.Println("config path: ", config_path)
			fmt.Println("=================================")
			uuid := fakers.NewUUIDFaker()


			uuid.Generate()
			fmt.Println(uuid.GetType())
			fmt.Println(uuid.GetFormat())

			test := [2]string{"PhoneFaker", "EmailFaker"}


			for i := 0 i < len(test) i++ {
				//basic factory pattern that will be iterated over for generation
				// Retrieve the correct faker implementation by name
				faker, found := fakers.GetFakerByName(test[i])
				if found {
					switch f := faker.(type) {
					case *fakers.PhoneFaker:
						// Call Generate on PhoneFaker
						f.Generate()
					case *fakers.EmailFaker:
						// Call Generate on EmailFaker
						f.Generate()
					default:
						fmt.Printf("Unknown faker type for: %s\n", test[i])
					}
				} else {
					fmt.Printf("Faker not found: %s\n", test[i])
				}
			}
			// email.Generate()
			// phone.Generate()
			// // print(json.ToJSONString(config))
			// // print(json.ToJSONString(config.Entities[0]))
		}

		if(scaffold && scaffold_name != ""){
			fmt.Println(scaffold)
			fmt.Println("generating faker..")
			fmt.Println("scaffold_name: ", scaffold_name)

			faker_config := FakerConfig{
				Name: scaffold_name,
				DataType: scaffold_name,
				Format: "",
			}
			GenerateFaker(faker_config)
		}

	},
}

func init() {
	//main flags
	rootCmd.PersistentFlags().Bool("version", false, "show cli version")
	rootCmd.PersistentFlags().Bool("verbose", false, "show additional logs")
	rootCmd.PersistentFlags().StringVar(&config_path, "config", "", "path to config file")

	//faker generation
	rootCmd.PersistentFlags().Bool("scaffold", false, "generate new faker scaffold")
	rootCmd.PersistentFlags().StringVar(&scaffold_name, "scaffold_name", "", "name of new faker")

}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute() err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
