package version

import (
	"context"
	"runtime"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/rkrmr33/template-hub/common"
	versionpkg "github.com/rkrmr33/template-hub/pkg/api/version"
)

type Server struct {
	versionpkg.UnimplementedVersionServiceServer
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Version(context.Context, *emptypb.Empty) (*versionpkg.VersionResponse, error) {
	return &versionpkg.VersionResponse{
		Version:   common.Version,
		GitCommit: common.GitCommit,
		BuildDate: common.BuildDate,
		GoVersion: runtime.Version(),
		Compiler:  runtime.Compiler,
	}, nil
}
