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
  repo, err := database.NewRepository("./database.db")
  if err != nil {
    log.Fatalf("Failed to connect: %v", err)
  }
  defer repo.Close()

  ratingsService := service.NewRatingsService(repo)

  lis, err := net.Listen("tcp", ":50051")
  if err != nil {
    log.Fatalf("Failed to listen: %v", err)
  }

  s := grpc.NewServer()
  pb.RegisterServiceServer(s, ratingsService)
  reflection.Register(s)

  log.Println("Server starting on :50051")
  if err := s.Serve(lis); err != nil {
    log.Fatalf("Failed to serve: %v", err)
  }
}
