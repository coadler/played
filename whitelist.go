package played

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"

	"github.com/coadler/played/pb"
)

func (s *Server) AddUser(ctx context.Context, req *pb.AddUserRequest) (*pb.AddUserResponse, error) {
	err := s.rdb.Set(fmtWhitelistKey(req.User), 1, 0).Err()
	if err != nil {
		s.log.Error("failed to add user to whitelist", zap.Error(err))
		return &pb.AddUserResponse{}, grpc.Errorf(codes.Internal, err.Error())
	}

	return &pb.AddUserResponse{}, nil
}

func (s *Server) RemoveUser(ctx context.Context, req *pb.RemoveUserRequest) (*pb.RemoveUserResponse, error) {
	err := s.rdb.Del(fmtWhitelistKey(req.User)).Err()
	if err != nil {
		s.log.Error("failed to delete whitelist", zap.Error(err))
		return &pb.RemoveUserResponse{}, grpc.Errorf(codes.Internal, err.Error())
	}

	return &pb.RemoveUserResponse{}, nil
}

func (s *Server) CheckWhitelist(ctx context.Context, req *pb.CheckWhitelistRequest) (*pb.CheckWhiteListResponse, error) {
	res, err := s.rdb.Exists(fmtWhitelistKey(req.User)).Result()
	if err != nil {
		s.log.Error("failed to read whitelist", zap.Error(err))
		return &pb.CheckWhiteListResponse{}, grpc.Errorf(codes.Internal, err.Error())
	}

	return &pb.CheckWhiteListResponse{
		Whitelisted: res == 1,
	}, nil
}
