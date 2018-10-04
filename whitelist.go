package played

import (
	"context"

	"github.com/coadler/played/pb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

func (s *PlayedServer) AddUser(ctx context.Context, req *pb.AddUserRequest) (*pb.AddUserResponse, error) {
	s.log.Info("got whitelist", zap.String("user", req.User))

	err := s.Redis.Set(fmtWhitelistKey(req.User), 1, 0).Err()
	if err != nil {
		s.log.Error("failed to add user to whitelist", zap.Error(err))
		return &pb.AddUserResponse{}, grpc.Errorf(codes.Internal, err.Error())
	}

	return &pb.AddUserResponse{}, nil
}

func (s *PlayedServer) RemoveUser(ctx context.Context, req *pb.RemoveUserRequest) (*pb.RemoveUserResponse, error) {
	s.log.Info("got remove", zap.String("user", req.User))

	err := s.Redis.Del(fmtWhitelistKey(req.User)).Err()
	if err != nil {
		s.log.Error("failed to delete whitelist", zap.Error(err))
		return &pb.RemoveUserResponse{}, grpc.Errorf(codes.Internal, err.Error())
	}

	return &pb.RemoveUserResponse{}, nil
}

func (s *PlayedServer) CheckWhitelist(ctx context.Context, req *pb.CheckWhitelistRequest) (*pb.CheckWhiteListResponse, error) {
	s.log.Info("got whitelist check", zap.String("user", req.User))

	res, err := s.Redis.Exists(fmtWhitelistKey(req.User)).Result()
	if err != nil {
		s.log.Error("failed to read whitelist", zap.Error(err))
		return &pb.CheckWhiteListResponse{}, grpc.Errorf(codes.Internal, err.Error())
	}

	return &pb.CheckWhiteListResponse{
		Whitelisted: res == 1,
	}, nil
}
