package engine

import (
	"math"
	"slices"
	"time"

	agents "simuresto/SimuResto/Agents"
	environment "simuresto/SimuResto/Environment"
)

const companionLaneOffset = 14

func queueJumpLimitDistance(world *environment.Environment) float64 {
	return routeLength(world.Starter1Route[:environment.Starter1QueueJumpLimitIndex+1])
}

func (e *engine) startCompanionWalk(world *environment.Environment, student *agents.StudentState) bool {
	companion := student.Student.Companion
	if companion.Role != agents.InvitedCompanion {
		return false
	}
	leader := world.StudentsByID[companion.LeaderID]
	if leader == nil || leader.Student.Companion.Role != agents.QueueJumpLeader {
		return false
	}
	limitDistance := queueJumpLimitDistance(world)
	if _, leaderInRange := queueJumpLeaderInRange(world, world.Queue.IDs(), companion.LeaderID, limitDistance); !leaderInRange {
		return false
	}

	student.Status = agents.Walking
	student.Target = agents.TargetJoinCompanion
	if len(e.sideLaneCompanions) > 0 {
		lastCompanion := e.sideLaneCompanions[len(e.sideLaneCompanions)-1]
		student.PositionAlongRoute = min(0, lastCompanion.PositionAlongRoute-world.Settings.QueueSpacing)
	}
	setCompanionLanePosition(world, student, student.PositionAlongRoute)
	e.sideLaneCompanions = append(e.sideLaneCompanions, student)
	return true
}

func queueJumpLeaderInRange(world *environment.Environment, queueIDs []int, leaderID int, limitDistance float64) (int, bool) {
	leaderIndex := slices.Index(queueIDs, leaderID)
	if leaderIndex < 0 {
		return -1, false
	}
	leader := world.StudentsByID[leaderID]
	return leaderIndex, leader.Target == agents.TargetStarter1 && leader.PositionAlongRoute <= limitDistance
}

// The queue-jump section of Starter1Route runs left and then down to the lower turn.
func queueJumpRoutePoint(world *environment.Environment, distance float64) (agents.Point, int) {
	entry, corner := world.Starter1Route[0], world.Starter1Route[1]
	cornerDistance := entry.X - corner.X
	if distance < 0 {
		return agents.Point{X: entry.X - distance, Y: entry.Y}, 0
	}
	if distance < cornerDistance {
		return agents.Point{X: entry.X - distance, Y: entry.Y}, 1
	}
	point := agents.Point{X: corner.X, Y: corner.Y + distance - cornerDistance}
	nextWaypoint := environment.Starter1DoorIndex
	for nextWaypoint <= environment.Starter1QueueJumpLimitIndex && point.Y >= world.Starter1Route[nextWaypoint].Y {
		nextWaypoint++
	}
	return point, nextWaypoint
}

func setCompanionLanePosition(world *environment.Environment, student *agents.StudentState, distance float64) {
	point, _ := queueJumpRoutePoint(world, distance)
	corner := world.Starter1Route[1]
	// Offset perpendicular to each straight leg and join the offsets at the corner.
	offsetX := max(0, companionLaneOffset-(point.X-corner.X))
	offsetY := max(0, companionLaneOffset-(point.Y-corner.Y))
	student.PositionAlongRoute = distance
	student.Position = agents.Point{X: point.X - offsetX, Y: point.Y - offsetY}
}

func (e *engine) advanceCompanionWalkers(world *environment.Environment, dt time.Duration, limitDistance float64) map[int]time.Duration {
	if len(e.sideLaneCompanions) == 0 {
		return nil
	}
	queueTimeLeft := make(map[int]time.Duration)
	companionAheadByLeader := make(map[int]*agents.StudentState)
	stillInSideLane := e.sideLaneCompanions[:0]
	for _, student := range e.sideLaneCompanions {
		leaderID := student.Student.Companion.LeaderID
		elapsed := min(dt, world.TimePassed-student.Student.ArrivalTime)
		if elapsed > 0 && advanceCompanionWalk(world, student, elapsed, limitDistance, queueTimeLeft, companionAheadByLeader[leaderID]) {
			continue
		}
		stillInSideLane = append(stillInSideLane, student)
		companionAheadByLeader[leaderID] = student
	}
	clear(e.sideLaneCompanions[len(stillInSideLane):])
	e.sideLaneCompanions = stillInSideLane
	return queueTimeLeft
}

func advanceCompanionWalk(world *environment.Environment, student *agents.StudentState, elapsed time.Duration,
	limitDistance float64, queueTimeLeft map[int]time.Duration, companionAhead *agents.StudentState,
) bool {
	queueIDs := world.Queue.IDs()
	leaderID := student.Student.Companion.LeaderID
	insertAfterIndex, leaderInRange := queueJumpLeaderInRange(world, queueIDs, leaderID, limitDistance)
	if !leaderInRange {
		student.Target = agents.TargetQueueTail
	}
	queueSpacing := world.Settings.QueueSpacing
	targetDistance := queueTailPosition(world, queueIDs)
	if student.Target == agents.TargetJoinCompanion {
		// Keep companions already in line together and preserve their generation order.
		for insertAfterIndex+1 < len(queueIDs) && world.StudentsByID[queueIDs[insertAfterIndex+1]].Student.Companion.LeaderID == leaderID {
			insertAfterIndex++
		}
		studentAhead := world.StudentsByID[queueIDs[insertAfterIndex]]
		targetDistance = studentAhead.PositionAlongRoute - queueSpacing
	}
	walkingDistance := world.Settings.WalkingSpeed * elapsed.Seconds()
	if companionAhead != nil && student.Target == agents.TargetJoinCompanion {
		targetDistance = min(targetDistance, companionAhead.PositionAlongRoute-queueSpacing)
	}
	currentDistance := student.PositionAlongRoute
	nextDistance := max(currentDistance-walkingDistance, min(targetDistance, currentDistance+walkingDistance))
	setCompanionLanePosition(world, student, nextDistance)
	if student.PositionAlongRoute != targetDistance {
		return false
	}
	if student.Target == agents.TargetQueueTail {
		if !world.Queue.Enqueue(student.Student.ID) {
			return false
		}
	} else {
		if companionAhead != nil {
			return false
		}
		if insertAfterIndex+1 < len(queueIDs) {
			studentBehind := world.StudentsByID[queueIDs[insertAfterIndex+1]]
			if studentBehind.PositionAlongRoute > targetDistance-queueSpacing+0.0001 {
				// Let the front move forward while this student leaves room for the merge.
				queueTimeLeft[studentBehind.Student.ID] = 0
				return false
			}
		}
		if !world.Queue.InsertAfter(queueIDs[insertAfterIndex], student.Student.ID) {
			return false
		}
	}
	// PositionAlongRoute reserves the slot while Position still follows the student.
	student.Target = agents.TargetQueueSlot
	student.NextWaypoint = 0
	sideWalkTime := time.Duration(math.Round(math.Abs(nextDistance-currentDistance) / world.Settings.WalkingSpeed * float64(time.Second)))
	queueTimeLeft[student.Student.ID] = max(0, elapsed-sideWalkTime)
	return true
}

func walkIntoQueueSlot(world *environment.Environment, student *agents.StudentState, dt time.Duration) time.Duration {
	slot, nextWaypoint := queueJumpRoutePoint(world, student.PositionAlongRoute)
	remaining := student.Walk([]agents.Point{slot}, world.Settings.WalkingSpeed, dt)
	if student.NextWaypoint == 1 {
		student.Target = agents.TargetStarter1
		student.NextWaypoint = nextWaypoint
	}
	return remaining
}
