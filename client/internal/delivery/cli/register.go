package cli

import (
	"context"
	"fmt"
	"log"
	"time"

	clientGRPC "gophKeeper/client/internal/transport/grpc"
	pb "gophKeeper/internal/proto"

	"github.com/spf13/cobra"
)

var registerCmd = &cobra.Command{
	Use:   "register",
	Short: "Register a new user",
	Long:  `Registers a new user on the GophKeeper server with a login and password.`,
	Run: func(cmd *cobra.Command, args []string) {
		login, _ := cmd.Flags().GetString("login")
		password, _ := cmd.Flags().GetString("password")

		if login == "" || password == "" {
			log.Fatal("Login and password are required")
		}

		serverAddr, _ := cmd.Flags().GetString("server")
		client, conn := clientGRPC.NewClient(serverAddr)
		defer conn.Close()

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		req := pb.RegisterRequest_builder{
			
			Login:    &login,
			Password: &password,
		}.Build()

		res, err := client.Register(ctx, req)
		if err != nil {
			log.Fatalf("could not register: %v", err)
		}

		// TODO auth
		// fmt.Printf("Registration successful! Your token: %s\n", res.Token)
		fmt.Println("Registration successful!",res.String())
		fmt.Println("Please save this token securely. It is required for future authentication.")
	},
}

func init() {
	registerCmd.Flags().StringP("login", "l", "", "User login (required)")
	registerCmd.Flags().StringP("password", "p", "", "User password (required)")
}
