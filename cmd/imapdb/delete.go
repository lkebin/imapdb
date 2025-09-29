package main

import (
	"log"
	"os"

	"github.com/lkebin/imapdb"
	"github.com/spf13/cobra"
)

// deleteCmd represents the delete command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a key-value pair by key",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := imapdb.NewDB(os.Getenv("IMAP_SERVER"), os.Getenv("IMAP_USER"), os.Getenv("IMAP_PASSWORD"))
		if err != nil {
			log.Fatal(err)
		}

		if err := db.Delete(args[0]); err != nil {
			log.Fatal(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
