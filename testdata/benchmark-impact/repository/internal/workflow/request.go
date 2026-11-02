package workflow

type Request struct {
	Amount int
	Status string
}

func SubmitRequest(amount int) Request {
	return Request{Amount: amount, Status: "SUBMITTED"}
}
