package agents

type Course string

const (
	Starter Course = "starter"
	Main    Course = "main"
	Dessert Course = "dessert"
)

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

	Abandoned StudentStatus = "abandoned"

	Walking          StudentStatus = "walking"
	Serving          StudentStatus = "receiving a dish"
	WalkingToCashier StudentStatus = "walking to checkout"
	Paying           StudentStatus = "paying"
	WalkingToDining  StudentStatus = "walking to dining"
	Dining           StudentStatus = "dining"
	WalkingToTrash   StudentStatus = "walking to waste station"
	WalkingToExit    StudentStatus = "walking to exit"
	Finished         StudentStatus = "finished"
)
