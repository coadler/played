package played

import (
	"context"

	"github.com/ThyLeader/played/pb"
	"github.com/boltdb/bolt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (s *PlayedServer) AddUser(ctx context.Context, req *pb.AddUserRequest) (*pb.AddUserResponse, error) {
	s.log.Info("got whitelist", zap.String("user", req.User))
	err := s.Bolt.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(s.WhitelistBucket).Put([]byte(req.User), []byte(""))
	})

	if err != nil {
		return &pb.AddUserResponse{}, grpc.Errorf(codes.Internal, err.Error())
	}

	return &pb.AddUserResponse{}, nil
}

func (s *PlayedServer) RemoveUser(ctx context.Context, req *pb.RemoveUserRequest) (*pb.RemoveUserResponse, error) {
	s.log.Info("got remove", zap.String("user", req.User))
	err := s.Bolt.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(s.WhitelistBucket).Delete([]byte(req.User))
	})
	if err != nil {
		return &pb.RemoveUserResponse{}, grpc.Errorf(codes.Internal, err.Error())
	}

	return &pb.RemoveUserResponse{}, nil
}

func (s *PlayedServer) CheckWhitelist(ctx context.Context, req *pb.CheckWhitelistRequest) (*pb.CheckWhiteListResponse, error) {
	s.log.Info("got whitelist check", zap.String("user", req.User))
	whitelisted := false
	err := s.Bolt.View(func(tx *bolt.Tx) error {
		whitelisted = tx.Bucket(s.WhitelistBucket).Get([]byte(req.User)) != nil
		return nil
	})
	if err != nil {
		return &pb.CheckWhiteListResponse{}, grpc.Errorf(codes.Internal, err.Error())
	}

	return &pb.CheckWhiteListResponse{
		Whitelisted: whitelisted,
	}, nil
}
