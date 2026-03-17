package main

import (
	"context"
	"log"
	"time"

	permissionpb "github.com/mrhumster/identity-service/gen/go/permission"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	client := permissionpb.NewPermissionServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	checkResp, err := client.CheckPermission(ctx, &permissionpb.CheckPermissionRequest{
		UserId:   "user123",
		Resource: "document",
		Action:   "read",
	})
	if err != nil {
		log.Fatalf("CheckPermission failed: %v", err)
	}
	log.Printf("✅ CheckPermission response: %v", checkResp.GetAllowed())

	addResp, err := client.AddPolicy(ctx, &permissionpb.AddPolicyRequest{
		Policy:     "user123",
		Resorce:    "document",
		Permission: "read",
	})
	if err != nil {
		log.Fatalf("AddPolicy failed: %v", err)
	}
	log.Printf("✅ AddPolicy response: %v", addResp.GetAdded())

	removeResp, err := client.RemovePolicy(ctx, &permissionpb.RemovePolicyRequest{
		Policy:     "user123",
		Resource:   "document",
		Permission: "read",
	})
	if err != nil {
		log.Fatalf("RemovePlicy failed: %v", err)
	}
	log.Printf("✅ RemovePolicy response: %v", removeResp.GetRemoved())

	addIfExistsResp, err := client.AddPolicyIfNotExists(ctx, &permissionpb.AddPolicyIfNotExistsRequest{
		Policy:     "user321",
		Resource:   "document",
		Permission: "write",
	})
	if err != nil {
		log.Fatalf("Add Policy if not exists failed: %v", err)
	}
	log.Printf("✅  Add Policy if not exists response %v", addIfExistsResp.GetExists())
}
