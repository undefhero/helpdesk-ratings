package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"helpdesk-ratings/internal/config"
	"helpdesk-ratings/internal/database"
	"helpdesk-ratings/internal/service"
	pb "helpdesk-ratings/proto/gen"
)

func main() {
	cfg := config.Load()

	log.Printf("Starting server with config: Port=%s, DB=%s", cfg.Server.Port, cfg.Database.FilePath)

	repo, err := database.NewRepository(cfg.Database.FilePath)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer repo.Close()

	ratingsService := service.NewRatingsService(repo)

	lis, err := net.Listen("tcp", ":"+cfg.Server.Port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterServiceServer(s, ratingsService)
	reflection.Register(s)

	log.Printf("Server starting on :%d", cfg.Server.Port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
