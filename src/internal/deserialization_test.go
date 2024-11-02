package internal

import (
	"math"
	"reflect"
	"testing"

	"github.com/GustavBW/bsc-multiplayer-backend/src/util"
)

// Not real data, will cause issues with anything else than deserialize.
// Simply just used for padding
var placeHolderIDAndEventIDBytes = []byte{
	0, 0, 0, 0, // Placeholder for user ID
	0, 0, 0, 0, // Event ID
}

func TestDeserializePlayerJoinedEvent(t *testing.T) {
	spec := PLAYER_JOINED_EVENT
	baseData := []byte{
		0, 0, 0, 42, // PlayerID
		72, 101, 108, 108, 111, // IGN: "Hello"
	}

	dataWHeader := util.CopyAndAppend(placeHolderIDAndEventIDBytes, baseData)
	expected := &PlayerJoinedMessageDTO{
		PlayerID: 42,
		IGN:      "Hello",
	}

	result1, err := Deserialize(spec, dataWHeader, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result1, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result1, expected)
	}

	result, err := Deserialize(spec, baseData, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializePlayerLeftEvent(t *testing.T) {
	spec := PLAYER_LEFT_EVENT
	data := []byte{
		0, 0, 0, 24, // PlayerID
		71, 111, 111, 100, 98, 121, 101, // IGN: "Goodbye"
	}
	expected := &PlayerLeftMessageDTO{
		PlayerID: 24,
		IGN:      "Goodbye",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializeEnterLocationEvent(t *testing.T) {
	spec := ENTER_LOCATION_EVENT
	data := []byte{
		0, 0, 0, 7, // Colony Location ID
	}
	expected := &EnterLocationMessageDTO{
		ID: 7,
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializePlayerMoveEvent(t *testing.T) {
	spec := PLAYER_MOVE_EVENT
	data := []byte{
		0, 0, 0, 10, // PlayerID
		0, 0, 0, 20, // ColonyLocationID
	}
	expected := &PlayerMoveMessageDTO{
		PlayerID:         10,
		ColonyLocationID: 20,
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializeDifficultySelectForMinigameEvent(t *testing.T) {
	spec := DIFFICULTY_SELECT_FOR_MINIGAME_EVENT
	data := []byte{
		0, 0, 0, 1, // MinigameID
		0, 0, 0, 2, // DifficultyID
		69, 97, 115, 121, // DifficultyName: "Easy"
	}
	expected := &DifficultySelectForMinigameMessageDTO{
		MinigameID:     1,
		DifficultyID:   2,
		DifficultyName: "Easy",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializePlayerReadyEvent(t *testing.T) {
	spec := PLAYER_READY_EVENT
	data := []byte{
		0, 0, 0, 33, // PlayerID
		82, 101, 97, 100, 121, 80, 108, 97, 121, 101, 114, // IGN: "ReadyPlayer"
	}
	expected := &PlayerReadyMessageDTO{
		PlayerID: 33,
		IGN:      "ReadyPlayer",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializeDifficultyConfirmedForMinigameEvent(t *testing.T) {
	spec := DIFFICULTY_CONFIRMED_FOR_MINIGAME_EVENT
	data := []byte{
		0, 0, 0, 1, // MinigameID
		0, 0, 0, 3, // DifficultyID
		72, 97, 114, 100, // DifficultyName: "Hard"
	}
	expected := &DifficultyConfirmedForMinigameMessageDTO{
		MinigameID:     1,
		DifficultyID:   3,
		DifficultyName: "Hard",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializePlayerAbortingMinigameEvent(t *testing.T) {
	spec := PLAYER_ABORTING_MINIGAME_EVENT
	data := []byte{
		0, 0, 0, 42, // PlayerID
		65, 98, 111, 114, 116, 101, 114, // IGN: "Aborter"
	}
	expected := &PlayerAbortingMinigameMessageDTO{
		PlayerID: 42,
		IGN:      "Aborter",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializePlayerJoinActivityEvent(t *testing.T) {
	spec := PLAYER_JOIN_ACTIVITY_EVENT
	data := []byte{
		0, 0, 0, 0, // Placeholder for user ID
		0, 0, 7, 214, // Event ID (2006 for PlayerJoinActivity)
		0, 0, 0, 55, // PlayerID
		74, 111, 105, 110, 101, 114, // IGN: "Joiner"
	}
	expected := &PlayerJoinActivityMessageDTO{
		PlayerID: 55,
		IGN:      "Joiner",
	}

	result, err := Deserialize(spec, data, false)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializeAsteroidSpawnEvent(t *testing.T) {
	spec := ASTEROID_SPAWN_EVENT
	floatYVal := util.BytesOfFloat32(300)
	data := []byte{
		0, 0, 0, 42, // ID: 42
		66, 200, 0, 0, // X: 100.0 (float32)
		floatYVal[0], floatYVal[1], floatYVal[2], floatYVal[3], // Y: 300.0 (float32 in Big Endian)
		5,          // Health: 5
		10,         // TimeUntilImpact: 10
		2,          // Type: 2
		65, 66, 67, // CharCode: "ABC"
	}
	expected := &AsteroidSpawnMessageDTO{
		ID:              42,
		X:               100.0,
		Y:               300.0,
		Health:          5,
		TimeUntilImpact: 10,
		Type:            2,
		CharCode:        "ABC",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Detailed comparison
	compareAsteroidSpawnDTO(t, result, expected)
}

func compareAsteroidSpawnDTO(t *testing.T, got, want *AsteroidSpawnMessageDTO) {
	t.Helper()
	if got.ID != want.ID {
		t.Errorf("ID mismatch: got %d, want %d", got.ID, want.ID)
	}
	if !almostEqual(got.X, want.X) {
		t.Errorf("X mismatch: got %f, want %f", got.X, want.X)
	}
	if !almostEqual(got.Y, want.Y) {
		t.Errorf("Y mismatch: got %f, want %f", got.Y, want.Y)
	}
	if got.Health != want.Health {
		t.Errorf("Health mismatch: got %d, want %d", got.Health, want.Health)
	}
	if got.TimeUntilImpact != want.TimeUntilImpact {
		t.Errorf("TimeUntilImpact mismatch: got %d, want %d", got.TimeUntilImpact, want.TimeUntilImpact)
	}
	if got.Type != want.Type {
		t.Errorf("Type mismatch: got %d, want %d", got.Type, want.Type)
	}
	if got.CharCode != want.CharCode {
		t.Errorf("CharCode mismatch: got %s, want %s", got.CharCode, want.CharCode)
	}
}

func almostEqual(a, b float32) bool {
	const epsilon = 1e-6
	return math.Abs(float64(a-b)) <= epsilon
}

func TestDeserializeAssignPlayerDataEvent(t *testing.T) {
	spec := ASSIGN_PLAYER_DATA_EVENT
	data := []byte{
		0, 0, 0, 1, // ID: 1
		65, 160, 0, 0, // X: 20.0 (float32)
		66, 32, 0, 0, // Y: 40.0 (float32)
		3,              // TankType: 3
		84, 65, 78, 75, // CharCode: "TANK"
	}
	expected := &AssignPlayerDataMessageDTO{
		ID:       1,
		X:        20.0,
		Y:        40.0,
		TankType: 3,
		CharCode: "TANK",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializeAsteroidImpactEvent(t *testing.T) {
	spec := ASTEROID_IMPACT_EVENT
	data := []byte{
		0, 0, 0, 5, // ID: 5
		0, 0, 3, 232, // ColonyHPLeft: 1000
	}
	expected := &AsteroidImpactOnColonyMessageDTO{
		ID:           5,
		ColonyHPLeft: 1000,
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}

func TestDeserializePlayerShootEvent(t *testing.T) {
	spec := PLAYER_SHOOT_EVENT
	data := []byte{
		0, 0, 0, 10, // PlayerID: 10
		83, 72, 79, 79, 84, // CharCode: "SHOOT"
	}
	expected := &PlayerShootAtCodeMessageDTO{
		PlayerID: 10,
		CharCode: "SHOOT",
	}

	result, err := Deserialize(spec, data, true)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Deserialized result does not match expected.\nGot: %+v\nWant: %+v", result, expected)
	}
}
