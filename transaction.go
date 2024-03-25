package main

type ActionType int

const (
	INSERT ActionType = iota
	DELETE
	DELETE_LINE
)

type Action struct {
	location   Location
	actionType ActionType
	text       string
	amount     int
}

type Transaction struct {
	actions  []Action
	location Location
}

func (t *Transaction) addAction(action Action) {
	t.actions = append([]Action{action}, t.actions...)
}

type Transactions struct {
	currentTransaction Transaction
	transactions       []Transaction
}

func NewTransactions() *Transactions {
	return &Transactions{
		currentTransaction: Transaction{},
		transactions:       make([]Transaction, 0),
	}
}

func (t *Transactions) submit(y, x int) {
	if len(t.currentTransaction.actions) == 0 {
		return
	}

	t.currentTransaction.location.col = x
	t.currentTransaction.location.line = y

	t.transactions = append(t.transactions, t.currentTransaction)
	t.currentTransaction = Transaction{}

	if len(t.transactions) > 100 {
		t.transactions = t.transactions[len(t.transactions)-100:]
	}
}

func (t *Transactions) addAction(action Action) {
	t.currentTransaction.addAction(action)
}

func (t *Transactions) pop() (bool, Transaction) {
	if len(t.transactions) == 0 {
		return false, Transaction{}
	}

	ta := t.transactions[len(t.transactions)-1]
	t.transactions = t.transactions[:len(t.transactions)-1]
	return true, ta
}
