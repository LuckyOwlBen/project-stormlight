package character

import (
	"fmt"
)

var sensesRangeTable = []int{5, 10, 20, 50, 100}
var liftingCapacityTable = []int{100, 200, 500, 1000, 5000, 10000}
var carryingCapacityTable = []int{50, 100, 250, 500, 2500, 5000}
var movementSpeedTable = []int{20, 25, 30, 40, 60, 80}
var recoveryDieTable = []string{"1d4", "1d6", "1d8", "1d10", "1d12", "1d20"}

func BuildDisplayObject(attributes Attributes) map[string]string {

	sensesRange := getSensesRange(attributes.Awareness)
	liftingCapacity := getLiftingCapacity(attributes.Strength)
	carryingCapacity := getCarryingCapacity(attributes.Strength)
	movementSpeed := getMovementSpeed(attributes.Speed)
	recoveryDie := getRecoveryDie(attributes.Willpower)

	return map[string]string{
		"Senses Range":      fmt.Sprintf("%d", sensesRange),
		"Lifting Capacity":  fmt.Sprintf("%d", liftingCapacity),
		"Carrying Capacity": fmt.Sprintf("%d", carryingCapacity),
		"Movement Speed":    fmt.Sprintf("%d", movementSpeed),
		"Recovery Die":      recoveryDie,
	}
}

func getSensesRange(awareness int) int {
	if awareness < 0 || awareness >= len(sensesRangeTable) {
		return 0
	}
	return sensesRangeTable[awareness]
}

func getLiftingCapacity(strength int) int {
	if strength < 0 || strength >= len(liftingCapacityTable) {
		return 0
	}
	return liftingCapacityTable[strength]
}

func getCarryingCapacity(strength int) int {
	if strength < 0 || strength >= len(carryingCapacityTable) {
		return 0
	}
	return carryingCapacityTable[strength]
}

func getMovementSpeed(speed int) int {
	if speed < 0 || speed >= len(movementSpeedTable) {
		return 0
	}
	return movementSpeedTable[speed]
}

func getRecoveryDie(willpower int) string {
	if willpower < 0 || willpower >= len(recoveryDieTable) {
		return "1d4"
	}
	return recoveryDieTable[willpower]
}
