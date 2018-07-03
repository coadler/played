package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/ThyLeader/played/pb"
	"github.com/bwmarrin/discordgo"
	"google.golang.org/grpc"
)

var (
	Token = ""
)

func main() {
	dg, err := discordgo.New("Bot " + Token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		panic(err.Error())
	}
	c := pb.NewPlayedClient(conn)

	dg.AddHandler(presenceSend(c))
	dg.AddHandler(messageCreate(c))
	dg.AddHandler(presenceSend2(c))
	dg.AddHandler(guildCreate(c))

	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}
	// start := time.Now()

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	dg.Close()
}

func presenceSend2(c pb.PlayedClient) func(s *discordgo.Session, m *discordgo.PresencesReplace) {
	ctx := context.Background()
	p, err := c.SendPlayed(ctx)
	if err != nil {
		fmt.Println(err)
		return func(s *discordgo.Session, m *discordgo.PresencesReplace) {}
	}

	return func(s *discordgo.Session, m *discordgo.PresencesReplace) {
		for _, e := range *m {
			name := ""
			if e.Game != nil {
				name = e.Game.Name
			}

			err := p.Send(&pb.SendPlayedRequest{
				User: e.User.ID,
				Game: name,
			})
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func presenceSend(c pb.PlayedClient) func(s *discordgo.Session, m *discordgo.PresenceUpdate) {
	ctx := context.Background()
	p, err := c.SendPlayed(ctx)
	if err != nil {
		fmt.Println(err)
		return func(s *discordgo.Session, m *discordgo.PresenceUpdate) {}
	}

	return func(s *discordgo.Session, m *discordgo.PresenceUpdate) {
		name := ""
		if m.Game != nil {
			name = m.Game.Name
		}

		err := p.Send(&pb.SendPlayedRequest{
			User: m.User.ID,
			Game: name,
		})
		if err != nil {
			fmt.Println(err)
		}
	}
}

func guildCreate(c pb.PlayedClient) func(s *discordgo.Session, g *discordgo.GuildCreate) {
	ctx := context.Background()
	p, err := c.SendPlayed(ctx)
	if err != nil {
		fmt.Println(err)
		return func(s *discordgo.Session, g *discordgo.GuildCreate) {}
	}

	return func(s *discordgo.Session, g *discordgo.GuildCreate) {
		if g.Unavailable {
			return
		}

		for _, e := range g.Presences {
			name := ""
			if e.Game != nil {
				name = e.Game.Name
			}

			err := p.Send(&pb.SendPlayedRequest{
				User: e.User.ID,
				Game: name,
			})
			if err != nil {
				fmt.Println(err)
			}
		}
	}
}

func messageCreate(c pb.PlayedClient) func(s *discordgo.Session, m *discordgo.MessageCreate) {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Content != "pls played" {
			return
		}

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
	}
}
