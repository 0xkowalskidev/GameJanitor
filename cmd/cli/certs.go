package cli

import (
	"fmt"

	"github.com/0xkowalskidev/gamejanitor/internal/tlsutil"
	"github.com/spf13/cobra"
)

var genWorkerCertCmd = &cobra.Command{
	Use:   "gen-worker-cert <worker-id>",
	Short: "Generate a TLS client certificate for a worker node",
	Long:  "Generates a worker certificate signed by the controller CA. Copy the CA cert, worker cert, and worker key to the worker node.",
	Args:  cobra.ExactArgs(1),
	RunE:  runGenWorkerCert,
}

func init() {
	genWorkerCertCmd.Flags().StringP("data-dir", "d", "/var/lib/gamejanitor", "Data directory (must match the controller's data-dir)")
	genWorkerCertCmd.GroupID = "server"
	rootCmd.AddCommand(genWorkerCertCmd)
}

func runGenWorkerCert(cmd *cobra.Command, args []string) error {
	dataDir, _ := cmd.Flags().GetString("data-dir")
	workerID := args[0]

	caCert, caKey, err := tlsutil.LoadOrCreateCA(dataDir)
	if err != nil {
		return fmt.Errorf("loading CA: %w", err)
	}

	certPath, keyPath, caPath, err := tlsutil.GenerateWorkerCert(dataDir, workerID, caCert, caKey)
	if err != nil {
		return fmt.Errorf("generating worker cert: %w", err)
	}

	fmt.Println("Worker certificate generated. Copy these files to the worker node:")
	fmt.Println()
	fmt.Printf("  CA cert:     %s\n", caPath)
	fmt.Printf("  Worker cert: %s\n", certPath)
	fmt.Printf("  Worker key:  %s\n", keyPath)
	fmt.Println()
	fmt.Println("On the worker, set:")
	fmt.Printf("  GJ_GRPC_CA=%s\n", caPath)
	fmt.Printf("  GJ_GRPC_CERT=%s\n", certPath)
	fmt.Printf("  GJ_GRPC_KEY=%s\n", keyPath)

	return nil
}
