package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/yaien/ngrok"
	"github.com/yaien/p2p"
)

func init() {
	config, _ := os.UserConfigDir()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filepath.Join(config, "p2p"))
	viper.AddConfigPath("$HOME/.p2p")
	viper.AddConfigPath(".")
	viper.ReadInConfig()
}

func main() {
	cmd := root()
	cmd.AddCommand(monitor())
	cmd.Execute()
}

func root() *cobra.Command {

	cmd := &cobra.Command{
		Use: "p2p",
		RunE: func(cmd *cobra.Command, args []string) error {

			l, err := net.Listen("tcp", fmt.Sprintf(":%d", viper.GetInt("port")))
			if err != nil {
				return fmt.Errorf("failed at listen: %w", err)
			}

			p := p2p.New(p2p.Options{
				Addr:   "http://" + l.Addr().String(),
				Name:   viper.GetString("name"),
				Lookup: viper.GetStringSlice("lookup"),
			})

			if viper.GetBool("ngrok") {
				ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
				defer cancel()
				tnl, err := ngrok.Open(ctx, ngrok.Options{Addr: l.Addr().String(), AuthToken: viper.GetString("ngrok-authtoken")})
				if err != nil {
					log.Fatal(err)
				}
				p.SetCurrentAddr(tnl.Url())
				log.Println("ngrok tunnel listening on", tnl.Url())
				log.Println("ngrok agent listening on", tnl.AgentUrl())
				defer tnl.Close()
			}

			go func() {
				log.Println("server listening on", p.Addr())
				p2p.HttpHandle(p, http.DefaultServeMux)
				err := http.Serve(l, nil)
				if err != nil {
					log.Fatalf("failed initializing server: %s", err)
				}
			}()

			go p.Start()

			ctx, _ := signal.NotifyContext(cmd.Context(), os.Interrupt, os.Kill)
			<-ctx.Done()
			log.Println("received stop request")
			return nil
		},
	}

	flags := cmd.Flags()
	flags.IntP("port", "p", 3000, "use -p to especify the current localhost server's port")
	flags.String("ngrok-authtoken", "", "use --ngrok-authtoken to set the ngrok auth token")
	flags.Bool("ngrok", false, "use --ngrok to serve p2p on an ngrok tunnel")
	flags.StringSlice("lookup", []string{}, "use --lookup to set initial adresses to be scanned")
	flags.String("key", "", "use --key to set the p2p common's key")
	flags.String("name", "", "use --name to set the current client's name")
	viper.BindPFlags(flags)

	return cmd
}

func monitor() *cobra.Command {
	return &cobra.Command{
		Use:  "monitor [addr]",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			monitor := p2p.NewMonitor(args[0])
			monitor.SetContext(cmd.Context())
			program := tea.NewProgram(monitor, tea.WithContext(cmd.Context()))
			program.Run()
		},
	}
}
