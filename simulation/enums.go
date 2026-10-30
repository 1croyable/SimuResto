package simulation

// Preference 素食还是正常？
type Preference string

const (
	Vegetarian    Preference = "vegetarian"
	MeatPreferred Preference = "meat_preferred"
)

// 菜品类别，前菜主菜甜点
type Course string

const (
	Starter Course = "starter"
	Main    Course = "main"
	Dessert Course = "dessert"
)

// Area 表示学生所在的物理区域。
type Area string

const (
	NoArea      Area = ""              // 尚未入场，或已经离开服务路线。
	Outside     Area = "outdoor_queue" //室外
	Inside      Area = "indoor_queue"  //Crous内部等候区
	SelfService Area = "self_service"  //取前菜和甜点的位置
	MainService Area = "main_service"  //拿主菜
	Payment     Area = "payment"       //付款的，原则上开1个或2个
)

// ServiceRoute 路线副本，调用方不能修改模拟内部的服务顺序。
func ServiceRoute() [5]Area {
	return [5]Area{Outside, Inside, SelfService, MainService, Payment}
}

// IsWaitingArea 判断是否处于领取托盘前的室内外队列。
func (a Area) IsWaitingArea() bool { return a == Outside || a == Inside }

// Status 表示学生生命周期，与所在区域分开。
type Status string

const (
	Scheduled Status = "scheduled"
	Active    Status = "active"
	Completed Status = "completed"
	Abandoned Status = "abandoned"
	Rejected  Status = "rejected"
	Cancelled Status = "cancelled"
)

// Signal 表示智能体交互或模拟状态变化的信号。
type Signal string

const (
	Arrived          Signal = "arrived"
	Entered          Signal = "entered"
	SelfServiceStart Signal = "self_service_started"
	FoodCollected    Signal = "food_collected"
	OrderPlaced      Signal = "order_placed"
	MainServed       Signal = "main_served"
	FoodUnavailable  Signal = "food_unavailable"
	PaymentStarted   Signal = "payment_started"
	Paid             Signal = "paid"
	Finished         Signal = "finished"
	LeftQueue        Signal = "left_queue"
	QueueFull        Signal = "queue_full"
	Stopped          Signal = "stopped"
	Progress         Signal = "progress"
)

// Role 表示工作人员岗位。
type Role string

const (
	Server  Role = "server"
	Cashier Role = "cashier"
)
