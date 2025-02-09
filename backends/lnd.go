package backends

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/lightningnetwork/lnd/lnrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

// LND contains all values needed to be able to connect to a node
type LND struct {
	GRPCHost     string `long:"grpchost" Description:"Host of the gRPC interface of LND"`
	CertFile     string `long:"certfile" Description:"TLS certificate for the LND gRPC and REST services"`
	MacaroonFile string `long:"macaroonfile" Description:"Macaroon file for authentication. Set to an empty string for no macaroon"`

	ctx    context.Context
	client lnrpc.LightningClient
}

// Connect to a node
func (lnd *LND) Connect() error {
	creds, err := credentials.NewClientTLSFromFile(lnd.CertFile, "")
	fmt.Println(lnd.CertFile)
	if err != nil {
		log.Error("Failed to read certificate for LND gRPC")

		return err
	}

	con, err := grpc.Dial(lnd.GRPCHost, grpc.WithTransportCredentials(creds))

	if err != nil {
		log.Error("Failed to connect to LND gRPC server")

		return err
	}

	if lnd.ctx == nil {
		lnd.ctx = context.Background()

		if lnd.MacaroonFile != "" {
			macaroon, err := getMacaroon(lnd.MacaroonFile)

			if macaroon == nil && err != nil {
				log.Error("Failed to read macaroon file of LND: ", err.Error())

			} else {
				lnd.ctx = metadata.NewOutgoingContext(lnd.ctx, macaroon)
			}

		}

	}

	lnd.client = lnrpc.NewLightningClient(con)

	return err
}

// GetInvoice gets and invoice from a node
func (lnd *LND) GetInvoice(message string, amount int64, expiry int64) (invoice string, rHash string, err error) {
	var response *lnrpc.AddInvoiceResponse

	response, err = lnd.client.AddInvoice(lnd.ctx, &lnrpc.Invoice{
		Memo:   message,
		Value:  amount,
		Expiry: expiry,
	})

	if err != nil {
		return "", "", err
	}

	return response.PaymentRequest, hex.EncodeToString(response.RHash), err
}

// InvoiceSettled checks if an invoice is settled by looking it up
func (lnd *LND) InvoiceSettled(rHash string) (settled bool, err error) {
	var invoice *lnrpc.Invoice

	rpcPaymentHash := lnrpc.PaymentHash{
		RHash: []byte(rHash),
	}

	invoice, err = lnd.client.LookupInvoice(lnd.ctx, &rpcPaymentHash)

	if err != nil {
		return false, err
	}

	return invoice.Settled, err
}

// TransactionSettled checks if an invoice is settled by looking it up
func (lnd *LND) TransactionSettled(rHash string) (settled bool, err error) {
	//var transactionDetails *lnrpc.Transaction

	/*rpcPaymentHash := lnrpc.PaymentHash{
		RHash: []byte(rHash),
	}*/
	/*rpcTransactionHash := lnrpc.GetTransactionsRequest{
			TxHash: rHash,
		}
	 // FFFFFFFFFFFFFFFFFFFIIIIIIIIIIIIIIIIIXXXXXXXXXXXXXXXXX
		transactionDetails, err = lnd.client.GetTransactions(lnd.ctx, &rpcTransactionHash)

		if err != nil {
			return false, err
		}

		return transactionDetails, err*/
	return true, err
}

// SubscribeInvoices subscribe to the invoice events of LND and calls a callback when one is settled
func (lnd *LND) SubscribeInvoices(publish PublishInvoiceSettled, rescan RescanPendingInvoices) error {
	stream, err := lnd.client.SubscribeInvoices(lnd.ctx, &lnrpc.InvoiceSubscription{})

	if err != nil {
		return err
	}

	wait := make(chan struct{})

	go func() {
		for {
			fmt.Println("ss")
			invoice, streamErr := stream.Recv()

			if streamErr == io.EOF {
				err = errors.New("lost connection to LND gRPC")

				close(wait)

				return
			}

			if streamErr != nil {
				err = streamErr

				close(wait)

				return
			}

			if invoice.Settled {
				go publish(invoice.PaymentRequest)
			}

		}

	}()

	// Connected successfully to LND
	// If there are pending invoices after reconnecting they should get rescanned now
	rescan()

	<-wait

	return err
}

// SubscribeTransactions subscribe to the invoice events of LND and calls a callback when one is settled
func (lnd *LND) SubscribeTransactions(publish PublishTransactionSettled, rescan RescanPendingTransactions) error {
	stream, err := lnd.client.SubscribeTransactions(lnd.ctx, &lnrpc.GetTransactionsRequest{})

	if err != nil {
		return err
	}

	wait := make(chan struct{})

	go func() {
		for {
			fmt.Println("ss")
			transaction, streamErr := stream.Recv()

			if streamErr == io.EOF {
				err = errors.New("lost connection to LND gRPC")

				close(wait)

				return
			}

			if streamErr != nil {
				err = streamErr

				close(wait)

				return
			}
			fmt.Printf("--%d--CONFIRMED OR NOT---", transaction.Amount)
			/*if transaction.Amount.Settled {
				go publish(transaction.PaymentRequest)
			}*/

		}

	}()

	// Connected successfully to LND
	// If there are pending invoices after reconnecting they should get rescanned now
	rescan()

	<-wait

	return err
}

// KeepAliveRequest is a dummy request to make sure the connection to LND doesn't time out if
// LND and LightningTip are separated with a firewall
func (lnd *LND) KeepAliveRequest() error {
	_, err := lnd.client.GetInfo(lnd.ctx, &lnrpc.GetInfoRequest{})

	return err
}

func getMacaroon(macaroonFile string) (macaroon metadata.MD, err error) {
	data, err := ioutil.ReadFile(macaroonFile)

	if err == nil {
		macaroon = metadata.Pairs("macaroon", hex.EncodeToString(data))
	}

	return macaroon, err
}
