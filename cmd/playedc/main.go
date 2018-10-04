package main

import (
	"context"
	"fmt"

	"github.com/coadler/played/pb"
	"google.golang.org/grpc"
)

func main() {
	conn, err := grpc.Dial("localhost:8089", grpc.WithInsecure())
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
                //
		// 	game++
		// 	time.Sleep(5 * time.Second)
		// }

		res, err := c.GetPlayed(ctx, &pb.GetPlayedRequest{User: "105484726235607040"})
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("first:", res.First)
		fmt.Println("last:", res.Last)
		for _, e := range res.Games {
			fmt.Printf("%+v\n", *e)
		}

		// res, err := c.CheckWhitelist(ctx, &pb.CheckWhitelistRequest{User: "105484726235607040"})
		// if err != nil {
		// 	fmt.Println(err)
		// 	return
		// }
                //
		// fmt.Println(res.Whitelisted)
	}()

	end := make(chan struct{})
	<-end
}
