package main

import (
	"fmt"
	"log"
	"os"

	"github.com/lkebin/imapdb"
	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a value by key",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := imapdb.NewDB(os.Getenv("IMAP_SERVER"), os.Getenv("IMAP_USER"), os.Getenv("IMAP_PASSWORD"))
		if err != nil {
			log.Fatal(err)
		}

		data, err := db.Get(args[0])
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%s", data)
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
