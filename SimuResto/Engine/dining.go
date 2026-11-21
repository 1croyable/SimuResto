package engine

import (
	"slices"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

func requestSeat(world *environment.Environment, student *agents.StudentState, from int, approach []agents.Point) {
	student.Status = agents.Walking
	student.Target = agents.TargetSeatWait
	student.DiningRoute = approach
	student.DiningNode = from
	student.NextWaypoint = 0
	world.SeatQueue.Enqueue(student.Student.ID)
	assignNextSeat(world)
}

// The simulation step is the sole writer. Assign the owner before sending the student to the seat.
func assignNextSeat(world *environment.Environment) {
	id, waiting := world.SeatQueue.Front()
	if !waiting || !slices.Contains(world.SeatOwners, 0) {
		return
	}
	student := world.StudentsByID[id]
	search := world.Dining.Search(student.DiningNode)
	index := preferredSeat(world, search)
	if index < 0 {
		return
	}
	world.SeatQueue.Dequeue()
	world.SeatOwners[index] = id
	student.SeatID = index + 1
	routeToSeat(world, student, search)
}

func preferredSeat(world *environment.Environment, search environment.DiningSearch) int {
	occupiedTables := make(map[int]bool)
	for index, owner := range world.SeatOwners {
		if owner != 0 {
			occupiedTables[world.Dining.Seats[index].Table] = true
		}
	}
	best, bestPriority, bestDistance := -1, 3, 0
	for index, seat := range world.Dining.Seats {
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

func routeToSeat(world *environment.Environment, student *agents.StudentState, search environment.DiningSearch) {
	seat := world.Dining.Seats[student.SeatID-1]
	student.DiningRoute = world.Dining.AppendRoute(student.DiningRoute, search, seat.Node)
	student.DiningRoute = append(student.DiningRoute, seat.Access, seat.Position)
	student.Target = agents.TargetDining
}

func leaveSeatForTrayReturn(world *environment.Environment, student *agents.StudentState) {
	index := student.SeatID - 1
	seat := world.Dining.Seats[index]
	world.SeatOwners[index] = 0
	student.SeatID = 0
	student.Status = agents.Walking
	student.Target = agents.TargetTrash
	student.NextWaypoint = 0
	approach := []agents.Point{seat.Access, world.Dining.Point(seat.Node)}
	student.DiningRoute = world.Dining.AppendRoute(approach, world.Dining.Search(seat.Node), world.Dining.TrayReturn)
	assignNextSeat(world)
}
