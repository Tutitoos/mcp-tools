package cli

import "github.com/spf13/cobra"

var nvidiaToolkitCmd = &cobra.Command{
	Use:   "nvidia-toolkit",
	Short: "Gestiona NVIDIA Container Toolkit (install / status / uninstall)",
	Long:  "Upgrade se hace vía `apt-get upgrade nvidia-container-toolkit` a mano — el subcomando 'upgrade' no está expuesto.",
}

func init() {
	nvidiaToolkitCmd.AddCommand(
		&cobra.Command{Use: "install", RunE: makeToolAction("nvidia-toolkit", "install")},
		&cobra.Command{Use: "uninstall", RunE: makeToolAction("nvidia-toolkit", "uninstall")},
		&cobra.Command{Use: "status", RunE: makeToolStatus("nvidia-toolkit")},
	)
	rootCmd.AddCommand(nvidiaToolkitCmd)
}
