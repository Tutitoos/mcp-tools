package cli

import (
	"github.com/spf13/cobra"

	"github.com/Tutitoos/mcp-tools/internal/docker"
)

var (
	logsFollow bool
	logsTail   int
)

var logsCmd = &cobra.Command{
	Use:   "logs <servicio>",
	Short: "Muestra logs de un servicio (--follow para seguir en tiempo real)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if logsFollow {
			return docker.Run("logs", "-f", args[0])
		}
		return docker.Run("logs", "--tail", intToString(logsTail), args[0])
	},
}

func intToString(n int) string {
	// Small formatter; strconv.Itoa avoided to keep imports minimal here.
	if n == 0 {
		return "0"
	}
	sign := ""
	if n < 0 {
		sign = "-"
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return sign + string(buf[i:])
}

func init() {
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "seguir logs en tiempo real")
	logsCmd.Flags().IntVar(&logsTail, "tail", 100, "líneas iniciales a mostrar (ignorado con --follow)")
	rootCmd.AddCommand(logsCmd)
}
