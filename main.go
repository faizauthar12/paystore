package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/faizauthar12/paystore/config"
	"github.com/faizauthar12/paystore/lib/helper"
	"github.com/faizauthar12/paystore/operation"
	pb "github.com/faizauthar12/paystore/protos"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	_ "github.com/lib/pq"
)

func main() {
	writeDB := helper.CreatePostgresConnection(
		os.Getenv("DB_WRITE_HOST"), os.Getenv("DB_WRITE_PORT"), os.Getenv("DB_WRITE_USER"),
		os.Getenv("DB_WRITE_PASSWORD"), os.Getenv("DB_WRITE_NAME"), os.Getenv("DB_WRITE_SSLMODE"))
	defer writeDB.Close()
	readDB := helper.CreatePostgresConnection(
		os.Getenv("DB_READ_HOST"), os.Getenv("DB_READ_PORT"), os.Getenv("DB_READ_USER"),
		os.Getenv("DB_READ_PASSWORD"), os.Getenv("DB_READ_NAME"), os.Getenv("DB_READ_SSLMODE"))
	defer readDB.Close()
	redis := helper.ConnectRedis(os.Getenv("REDIS_HOST"), os.Getenv("REDIS_USER"),
		os.Getenv("REDIS_PASS"), false)

	config := config.DefaultConfig(os.Getenv("PAYMENT_VENDOR_TABLE_NAME"), os.Getenv("WITHDRAW_VENDOR_TABLE_NAME"))

	// GRPC Setup
	paystoreClient := operation.New(writeDB, readDB, redis, config)
	grpcServer := grpc.NewServer()
	grpcHandler := operation.NewGRPCHandler(paystoreClient)
	pb.RegisterPaystoreServer(grpcServer, grpcHandler)
	reflection.Register(grpcServer)

	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	go func() {
		log.Printf("gRPC server listening on port %s", port)
		if err := grpcServer.Serve(listener); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

	app := fiber.New()

	app.Listen(":" + os.Getenv("PORT"))
}
