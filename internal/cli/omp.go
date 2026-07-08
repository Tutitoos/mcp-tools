package cli

import "github.com/spf13/cobra"

var ompCliCmd = &cobra.Command{
	Use:   "omp",
	Short: "Gestiona OMP CLI (curl https://omp.sh/install.sh | bash)",
}

func init() {
	ompCliCmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("omp", "install")},
		&cobra.Command{Use: "upgrade", RunE: makeToolAction("omp", "upgrade")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("omp", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("omp")},
	)
	rootCmd.AddCommand(ompCliCmd)
}
