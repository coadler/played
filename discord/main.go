package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"google.golang.org/grpc"

	"github.com/coadler/played/pb"
)

const Token = "Bot ..."

func main() {
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	conn, err := grpc.Dial("localhost:8089", grpc.WithInsecure())
	if err != nil {
		panic(err.Error())
	}
	c := pb.NewPlayedClient(conn)

	dg.AddHandler(messageCreate(c))

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func messageCreate(c pb.PlayedClient) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Content == "pls played" {
			ctx := context.Background()
			res, err := c.GetPlayed(ctx, &pb.GetPlayedRequest{User: m.Author.ID})
			if err != nil {
				fmt.Println(err)
				s.ChannelMessageSend(m.ChannelID, err.Error())
			}

			buf := new(strings.Builder)
			for _, e := range res.Games {
				buf.WriteString(fmt.Sprintf("%+v\n", e))
			}

			s.ChannelMessageSend(m.ChannelID, buf.String())
			return
		}

		if m.Content == "pls whitelist" {
			ctx := context.Background()
			_, err := c.AddUser(ctx, &pb.AddUserRequest{User: m.Author.ID})
			if err != nil {
				fmt.Println(err)
				s.ChannelMessageSend(m.ChannelID, err.Error())
			}

			s.ChannelMessageSend(m.ChannelID, ":ok_hand:")
		}

	}
}
