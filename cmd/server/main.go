package main

import (
  "log"
  "net"

  "google.golang.org/grpc"
  "google.golang.org/grpc/reflection"
  pb "helpdesk-ratings/proto/gen"
  "helpdesk-ratings/internal/database"
  "helpdesk-ratings/internal/service"
)

func main() {
  cfg, err := config.Load()
  if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
  }
  
  log.Printf("Starting server with config: Port=%s, DB=%s", cfg.Server.Port, cfg.Database.FilePath)

  repo, err := database.NewRepository(cfg.Database.FilePath)
  if err != nil {
    log.Fatalf("Failed to connect: %v", err)
  }
  defer repo.Close()

  ratingsService := service.NewRatingsService(repo)

  lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Server.Port))
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
