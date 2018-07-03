package main

import (
	"context"
	"fmt"

	"github.com/ThyLeader/played/pb"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure())
	if err != nil {
		panic(err.Error())
	}
	c := pb.NewPlayedClient(conn)

	go func() {
		// ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		// defer cancel()
		ctx := context.Background()

		// p, _ := c.SendPlayed(ctx)
		// game := 0
		// for game < 5 {
		// 	fmt.Println("sent request")
		// 	p.Send(&pb.SendPlayedRequest{
		// 		User: "105484726235607040",
		// 		Game: fmt.Sprintf("%d", game),
		// 	})

		// 	game++
		// 	time.Sleep(5 * time.Second)
		// }

		res, _ := c.GetPlayed(ctx, &pb.GetPlayedRequest{User: "454072114492866560"})
		for _, e := range res.Games {
			fmt.Printf("%+v\n", *e)
		}
	}()

	end := make(chan struct{})
	<-end
}
