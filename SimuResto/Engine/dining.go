package engine

import (
	"math"
	"slices"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func (e *engine) prepareDining(world *environment.Environment, student *agents.StudentState) {
	durationOwner := student
	if student.Student.Companion.Role == agents.InvitedCompanion {
		if leader := world.StudentsByID[student.Student.Companion.LeaderID]; leader != nil {
			durationOwner = leader
		}
	}

	// The first group member to pay can initialize the leader's planned duration.
	if durationOwner.DiningTime == 0 {
		decision := e.requestStudentDecision(durationOwner, agents.StudentObservation{
			DecisionType:  agents.DecideDiningTime,
			MinDiningTime: world.Settings.MinDiningTime,
			MaxDiningTime: world.Settings.MaxDiningTime,
		})
		durationOwner.DiningTime = decision.DiningTime
	}
	student.DiningTime = durationOwner.DiningTime
	student.RemainingDining = student.DiningTime
}

func (e *engine) requestSeat(world *environment.Environment, student *agents.StudentState, from int, approach []agents.Point) {
	student.Status = agents.Walking
	student.Target = agents.TargetSeatWait
	student.DiningRoute = approach
	student.DiningNode = from
	student.NextWaypoint = 0
	world.SeatQueue.Enqueue(student.Student.ID)
	e.assignNextSeat(world)
}

// The simulation step is the sole writer. Assign the owner before sending the student to the seat.
func (e *engine) assignNextSeat(world *environment.Environment) {
	id, waiting := world.SeatQueue.Front()
	if !waiting || !slices.Contains(world.SeatOwners, 0) {
		return
	}
	student := world.StudentsByID[id]
	search := e.dining.Search(student.DiningNode)
	index := e.preferredSeat(world, student, search)
	if index < 0 {
		return
	}
	world.SeatQueue.Dequeue()
	world.SeatOwners[index] = id
	student.SeatID = index + 1
	e.routeToSeat(student, search)
}

func (e *engine) preferredSeat(world *environment.Environment, student *agents.StudentState, search DiningSearch) int {
	if groupID := companionGID(student.Student); groupID != 0 {
		var groupSeats []DiningSeat
		for index, id := range world.SeatOwners {
			if owner := world.StudentsByID[id]; owner != nil && id != student.Student.ID &&
				companionGID(owner.Student) == groupID {
				groupSeats = append(groupSeats, e.dining.Seats[index])
			}
		}
		if len(groupSeats) > 0 {
			return e.seatNearGroup(world, groupSeats, search)
		}
	}

	occupiedTables := make(map[int]bool)
	for index, owner := range world.SeatOwners {
		if owner != 0 {
			occupiedTables[e.dining.Seats[index].Table] = true
		}
	}
	best, bestPriority, bestDistance := -1, 3, 0
	for index, seat := range e.dining.Seats {
		distance := search.Distance[seat.Node]
		if world.SeatOwners[index] != 0 || distance < 0 {
			continue
		}
		priority := 0
		if occupiedTables[seat.Table] {
			priority = 1
			for _, neighbor := range seat.Neighbors {
				if world.SeatOwners[neighbor] != 0 {
					priority = 2
					break
				}
			}
		}
		if priority < bestPriority || priority == bestPriority && distance < bestDistance {
			best, bestPriority, bestDistance = index, priority, distance
		}
	}
	return best
}

func companionGID(student agents.StudentData) int {
	switch student.Companion.Role {
	case agents.TogetherLeader, agents.QueueJumpLeader:
		return student.ID
	case agents.InvitedCompanion:
		return student.Companion.LeaderID
	default:
		return 0
	}
}

func (e *engine) seatNearGroup(world *environment.Environment, groupSeats []DiningSeat, search DiningSearch) int {
	best, bestPriority, bestWalk := -1, 2, 0
	bestDistance := 0.0
	for index, seat := range e.dining.Seats {
		walk := search.Distance[seat.Node]
		if world.SeatOwners[index] != 0 || walk < 0 {
			continue
		}
		for _, groupSeat := range groupSeats {
			priority := 0
			if seat.Table != groupSeat.Table {
				priority = 1
			}
			distance := math.Hypot(seat.Position.X-groupSeat.Position.X, seat.Position.Y-groupSeat.Position.Y)
			if priority < bestPriority ||
				priority == bestPriority && (distance < bestDistance || distance == bestDistance && walk < bestWalk) {
				best, bestPriority, bestDistance, bestWalk = index, priority, distance, walk
			}
		}
	}
	return best
}

func (e *engine) routeToSeat(student *agents.StudentState, search DiningSearch) {
	seat := e.dining.Seats[student.SeatID-1]
	student.DiningRoute = e.dining.AppendRoute(student.DiningRoute, search, seat.Node)
	student.DiningRoute = append(student.DiningRoute, seat.Access, seat.Position)
	student.Target = agents.TargetDining
}

func (e *engine) leaveSeatForTrayReturn(world *environment.Environment, student *agents.StudentState) {
	index := student.SeatID - 1
	seat := e.dining.Seats[index]
	world.SeatOwners[index] = 0
	student.SeatID = 0
	student.Status = agents.Walking
	student.Target = agents.TargetTrash
	student.NextWaypoint = 0
	approach := []agents.Point{seat.Access, e.dining.Point(seat.Node)}
	student.DiningRoute = e.dining.AppendRoute(approach, e.dining.Search(seat.Node), e.dining.TrayReturn)
	e.assignNextSeat(world)
}
