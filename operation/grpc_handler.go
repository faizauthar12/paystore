package operation

import (
	"context"

	"github.com/faizauthar12/paystore/lib/organization"
	"github.com/faizauthar12/paystore/lib/payment"
	"github.com/faizauthar12/paystore/lib/withdraw"
	pb "github.com/faizauthar12/paystore/protos"
)

// GRPCHandler
/*
	- CreateBalance
	- CreatePayment
	- FinalizedPayment
	- CreateWithdraw
	- FinalizedWithdraw
*/
func pbToGoPaymentStatus(pbStatus pb.PaymentStatus) payment.PaymentStatus {
	switch pbStatus {
	case pb.PaymentStatus_PAYMENT_STATUS_PENDING:
		return payment.PaymentStatusPending
	case pb.PaymentStatus_PAYMENT_STATUS_PAID:
		return payment.PaymentStatusPaid
	case pb.PaymentStatus_PAYMENT_STATUS_FAILED:
		return payment.PaymentStatusFailed
	default:
		return payment.PaymentStatusPending // or handle error
	}
}

func pbToGoWithdrawStatus(pbStatus pb.PaymentStatus) withdraw.WithdrawStatus {
	switch pbStatus {
	case pb.PaymentStatus_PAYMENT_STATUS_PENDING:
		return withdraw.StatusPending
	case pb.PaymentStatus_PAYMENT_STATUS_PAID:
		return withdraw.StatusSuccess
	case pb.PaymentStatus_PAYMENT_STATUS_FAILED:
		return withdraw.StatusFailed
	default:
		return withdraw.StatusPending // or handle error
	}
}

type GRPCHandler struct {
	pb.UnimplementedPaystoreServer
	paystoreClient *PaystoreClient
}

func (grpc *GRPCHandler) CreateBalance(ctx context.Context, in *pb.CreateBalanceRequest) (*pb.CreatedResponse, error) {
	balance, errCreate := grpc.paystoreClient.CreateBalance(in.ExternalID, in.Currency, in.OrganizationSlug)
	if errCreate != nil {
		if errCreate == organization.OrganizationNotFound {
			return nil, organization.OrganizationNotFound
		}
		return nil, errCreate
	}

	return &pb.CreatedResponse{ID: balance.GetUUID()}, nil
}

func (grpc *GRPCHandler) CreatePayment(ctx context.Context, in *pb.CreatePaymentRequest) (*pb.CreatedResponse, error) {
	payment, errCreate := grpc.paystoreClient.CreatePayment(in.AccountUUID, in.Amount)
	if errCreate != nil {
		return nil, errCreate
	}

	return &pb.CreatedResponse{ID: payment.GetUUID()}, nil
}

func (grpc *GRPCHandler) FinalizedPayment(ctx context.Context, in *pb.FinalizedPaymentRequest) (*pb.EmptyResponse, error) {
	errFinalized := grpc.paystoreClient.FinalizedPayment(in.AccountUUID, in.PaymentUUID, pbToGoPaymentStatus(in.PaymentStatus), in.VendorRecordId)
	if errFinalized != nil {
		return nil, errFinalized
	}

	return &pb.EmptyResponse{}, nil
}

func (grpc *GRPCHandler) CreateWithdraw(ctx context.Context, in *pb.CreateWithdrawRequest) (*pb.CreatedResponse, error) {
	withdraw, errCreate := grpc.paystoreClient.CreateWithdraw(in.AccountUUID, in.Amount)
	if errCreate != nil {
		return nil, errCreate
	}

	return &pb.CreatedResponse{ID: withdraw.GetUUID()}, nil
}

func (grpc *GRPCHandler) FinalizedWithdraw(ctx context.Context, in *pb.FinalizedWithdrawRequest) (*pb.EmptyResponse, error) {
	errFinalized := grpc.paystoreClient.FinalizedWithdraw(in.WithdrawUUID, in.WithdrawUUID, pbToGoWithdrawStatus(in.WithdrawStatus), in.VendorRecordId)
	if errFinalized != nil {
		return nil, errFinalized
	}

	return &pb.EmptyResponse{}, nil
}

func NewGRPCHandler(paystoreClient *PaystoreClient) *GRPCHandler {
	return &GRPCHandler{
		paystoreClient: paystoreClient,
	}
}
