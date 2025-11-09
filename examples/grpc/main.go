package main

import (
	"log"
	"log/slog"
	"net"
	"os"

	"github.com/user/vibrant-auth-middleware-go/jwtauth"
	"google.golang.org/grpc"
)

// This is a simple example demonstrating gRPC JWT authentication.
// In a real application, you would have proper protobuf definitions.

func main() {
	// For this demo, we'll use HS256. In production with gRPC microservices,
	// you'd typically use RS256 with public key distribution.
	secret := []byte("your-256-bit-secret-key-min-32-bytes-here-for-demo!")

	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := jwtauth.NewConfig(
		jwtauth.WithHS256(secret),
		jwtauth.WithLogger(logger),
	)
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	// Create gRPC server with JWT interceptor
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(jwtauth.UnaryServerInterceptor(cfg)),
	)

	// In a real application, you would register your gRPC services here
	// pb.RegisterUserServiceServer(srv, &userServiceImpl{})

	// Start server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Println("gRPC server starting on :50051")
	log.Println("JWT authentication enabled via UnaryServerInterceptor")
	log.Println("Add 'authorization: Bearer <token>' to gRPC metadata")

	if err := srv.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// Example service implementation (commented out since we don't have protobuf definitions)
/*
type userServiceImpl struct {
	pb.UnimplementedUserServiceServer
}

func (s *userServiceImpl) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
	// Retrieve validated JWT claims
	claims, ok := jwtauth.GetClaims(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "claims not found")
	}

	// Access claims
	userID := claims.Subject

	// Example: Authorization check
	role := ""
	if roleVal, ok := claims.Custom["role"]; ok {
		if roleStr, ok := roleVal.(string); ok {
			role = roleStr
		}
	}

	if role != "admin" && userID != req.Id {
		return nil, status.Errorf(codes.PermissionDenied, "insufficient permissions")
	}

	// Return user data
	return &pb.User{
		Id:    userID,
		Email: claims.Custom["email"].(string),
	}, nil
}
*/

// Example client code for testing
func exampleClient() {
	// In a real application, you would:
	// 1. Create a gRPC connection
	// conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	// defer conn.Close()

	// 2. Create a client
	// client := pb.NewUserServiceClient(conn)

	// 3. Add JWT to metadata
	// md := metadata.Pairs("authorization", "Bearer YOUR_JWT_TOKEN")
	// ctx := metadata.NewOutgoingContext(context.Background(), md)

	// 4. Make authenticated RPC call
	// resp, err := client.GetUser(ctx, &pb.GetUserRequest{Id: "123"})

	log.Println("Example client code - see comments in grpc/main.go")
}
