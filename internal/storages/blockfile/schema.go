package blockfile

const (
	// freezerHashTable indicates the name of the freezer canonical hash table.
	BlockFileHashTable = "hashes"

	// freezerBodiesTable indicates the name of the freezer block body table.
	BlockFileBodiesTable = "bodies"

	// freezerHeaderTable indicates the name of the freezer header table.
	BlockFileTXsTable = "transactions"

	// freezerReceiptTable indicates the name of the freezer receipts table.
	BlockFileReceiptTable = "receipts"

	// freezerReceiptTable indicates the name of the freezer receipts table.
	BlockFileInterchainTable = "interchain"
)

var BlockFileSchema = map[string]bool{
	BlockFileHashTable:       true,
	BlockFileBodiesTable:     true,
	BlockFileTXsTable:        true,
	BlockFileReceiptTable:    true,
	BlockFileInterchainTable: true,
}
