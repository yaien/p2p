package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
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
	viper.SetDefault("transport", "rest")

	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.p2p")
	viper.AddConfigPath(filepath.Join(config, "p2p"))
	viper.SetEnvPrefix("p2p")
	viper.AutomaticEnv()
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
				Addr:      viper.GetString("address"),
				Name:      viper.GetString("name"),
				Lookup:    viper.GetStringSlice("lookup"),
				Transport: &p2p.HttpTransport{Key: viper.GetString("key")},
			})

			if viper.Get("transport") == "rest" {
				go func() {
					log.Println("rest server listening on", p.CurrentAddr())
					key := viper.GetString("key")
					sub := p2p.NewSubscriber(p.Channel())
					srv := p2p.NewHttpServer(p, sub, key)
					err = srv.Serve(l)
					if err != nil {
						log.Fatalf("failed initializing server: %s", err)
					}
				}()
			}

			if viper.Get("transport") == "grpc" {
				go func() {
					p.SetTransport(&p2p.GrpcTransport{})
					log.Println("grpc server listening on", p.CurrentAddr())
					sub := p2p.NewSubscriber(p.Channel())
					srv := p2p.NewGrpcServer(p, sub)
					err = srv.Serve(l)
					if err != nil {
						log.Fatalf("failed initializing server: %s", err)
					}
				}()
			}

			if viper.GetBool("ngrok") {
				ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
				defer cancel()
				tnl, err := ngrok.Open(ctx, ngrok.Options{Addr: l.Addr().String(), AuthToken: viper.GetString("ngrok-authtoken")})
				if err != nil {
					log.Fatal(err)
				}
				url := tnl.Url()
				if viper.GetString("transport") == "grpc" {
					url = strings.TrimPrefix(url, "https://")
				}
				p.SetCurrentAddr(url)
				log.Println("ngrok tunnel listening on", tnl.Url())
				log.Println("ngrok agent listening on", tnl.AgentUrl())
				defer tnl.Close()
			}

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
	flags.StringSliceP("lookup", "l", []string{}, "use --lookup to set initial addresses to be scanned")
	flags.String("key", "", "use --key to set the p2p common's key")
	flags.String("name", "", "use --name to set the current client's name")
	flags.StringP("transport", "t", "rest", "use --transport [rest|grpc] to specify the current p2p transport")
	flags.StringP("address", "a", "", "use --address to specify the current p2p address")
	viper.BindPFlags(flags)

	return cmd
}

func monitor() *cobra.Command {
	var transport string
	cmd := &cobra.Command{
		Use:  "monitor [addr]",
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			subscribe := p2p.Subscribe2Http
			if transport == "grpc" {
				subscribe = p2p.Subscribe2Grpc
			}
			monitor := p2p.NewMonitor(args[0], subscribe)
			monitor.SetContext(cmd.Context())
			program := tea.NewProgram(monitor, tea.WithContext(cmd.Context()))
			program.Run()
			err := monitor.Error()
			if err != nil {
				log.Println(err)
			}
		},
	}

	cmd.Flags().StringVarP(&transport, "transport", "t", "rest", "--use n [rest|grpc] to specify the target monitor transport")
	return cmd
}
