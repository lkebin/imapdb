package main

import (
	"log"
	"os"

	"github.com/lkebin/imapdb"

	"github.com/spf13/cobra"
)

// setCmd represents the set command
var setCmd = &cobra.Command{
	Use:   "set",
	Short: "Set a key-value pair",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := imapdb.NewDB(os.Getenv("IMAP_SERVER"), os.Getenv("IMAP_USER"), os.Getenv("IMAP_PASSWORD"))
		if err != nil {
			log.Fatal(err)
		}

		if err := db.Set(args[0], []byte(args[1])); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(setCmd)
}
