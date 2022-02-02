package registry

import (
	"github.com/rkrmr33/pkg/log"
	regapi "github.com/rkrmr33/template-hub/pkg/api/registry"
)

type server struct {
	logger *log.Logger
	data   []byte
}

func NewServer(data []byte) regapi.RegistryServiceServer {
	return &server{
		logger: log.G().Named("registry-server"),
		data:   data,
	}
}

func (s *server) Pull(req *regapi.PullRequest, res regapi.RegistryService_PullServer) error {
	for i := 0; i <= len(s.data); i += 1024 {
		if len(s.data) <= i+1024 {
			_ = res.Send(&regapi.PullResponse{
				Chunk: s.data[i:],
			})
		} else {
			_ = res.Send(&regapi.PullResponse{
				Chunk: s.data[i : i+1024],
			})
		}
	}

	return nil
}
