package main

type ActionType int

const (
	INSERT ActionType = iota
	DELETE
	DELETE_LINE
	ADD_LINE
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
	undoIndex          int
}

func NewTransactions() *Transactions {
	return &Transactions{
		currentTransaction: Transaction{},
		transactions:       make([]Transaction, 0),
		undoIndex:          -1,
	}
}

func (t *Transactions) submit(y, x int) {
	if len(t.currentTransaction.actions) == 0 {
		return
	}

	t.currentTransaction.location.col = x
	t.currentTransaction.location.line = y

	t.transactions = append(t.transactions[:t.undoIndex+1], t.currentTransaction)
	t.currentTransaction = Transaction{}

	if len(t.transactions) > 100 {
		t.transactions = t.transactions[len(t.transactions)-100:]
	}

	t.undoIndex = len(t.transactions) - 1
}

func (t *Transactions) addAction(action Action) {
	t.currentTransaction.addAction(action)
}

func (t *Transactions) redoPop() (bool, Transaction) {
	if len(t.transactions) == 0 || t.undoIndex >= len(t.transactions)-1 {
		return false, Transaction{}
	}

	t.undoIndex++
	ta := t.transactions[t.undoIndex]
	return true, ta
}

func (t *Transactions) pop() (bool, Transaction) {
	if len(t.transactions) == 0 || t.undoIndex <= -1 {
		return false, Transaction{}
	}

	ta := t.transactions[t.undoIndex]
	t.undoIndex--
	return true, ta
}
