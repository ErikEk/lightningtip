package backends

// PublishInvoiceSettled is a callback for a settled invoice
type PublishInvoiceSettled func(invoice string)

// PublishTransactionSettled is a callback for a settled transaction
type PublishTransactionSettled func()

// RescanPendingInvoices is a callbacks when reconnecting
type RescanPendingInvoices func()

// RescanPendingTransactions is a callbacks when reconnecting
type RescanPendingTransactions func()

// Backend is an interface that would allow for different implementations of Lightning to be used as backend
type Backend interface {
	Connect() error

	// The amount is denominated in satoshis and the expiry in seconds
	GetInvoice(description string, amount int64, expiry int64) (invoice string, rHash string, err error)

	InvoiceSettled(rHash string) (settled bool, err error)

	//InvoiceTransaction(rHash string) (settled bool, err error)

	SubscribeInvoices(publish PublishInvoiceSettled, rescan RescanPendingInvoices) error

	SubscribeTransactions(publish PublishTransactionSettled, rescan RescanPendingTransactions) error

	KeepAliveRequest() error
}
