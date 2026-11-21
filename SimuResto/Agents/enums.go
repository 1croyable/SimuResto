package agents

type Role string

const (
	Server  Role = "server"
	Cashier Role = "cashier"
)

type StudentStatus string

const (
	Scheduled         StudentStatus = "scheduled"
	WaitingForFood    StudentStatus = "waiting for food"
	WaitingForPayment StudentStatus = "waiting for payment"
	WaitingForSauce   StudentStatus = "waiting for sauce"

	Abandoned StudentStatus = "abandoned"

	Walking  StudentStatus = "walking"
	Serving  StudentStatus = "receiving a dish"
	Paying   StudentStatus = "paying"
	Dining   StudentStatus = "dining"
	Finished StudentStatus = "finished"
)

type StudentTarget string

const (
	TargetStarter1 StudentTarget = "starter1"
	TargetStarter2 StudentTarget = "starter2"
	TargetMain     StudentTarget = "main"
	TargetCashier  StudentTarget = "cashier"
	TargetSauce    StudentTarget = "sauce"
	TargetDining   StudentTarget = "dining"
	TargetSeatWait StudentTarget = "waiting for seat"
	TargetTrash    StudentTarget = "trash"
	TargetExit     StudentTarget = "exit"
)
