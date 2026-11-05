package agents

type Preference string

const (
	Vegetarian    Preference = "vegetarian"
	MeatPreferred Preference = "meat_preferred"
)

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
