package player_test

import (
	"testing"

	"futbol-app/internal/domain/player"
)

// --- Tests de creación ---

func TestNew_ValidPlayer(t *testing.T) {
	p, err := player.New("Yuber", player.PositionDefensa, 9)

	if err != nil {
		t.Fatalf("esperaba nil error, got: %v", err)
	}
	if p.Name != "Yuber" {
		t.Errorf("esperaba nombre 'Yuber', got: %s", p.Name)
	}
	if p.PrimaryPosition != player.PositionDefensa {
		t.Errorf("esperaba posicion defensa, got: %s", p.PrimaryPosition)
	}
	if p.Rating != 9 {
		t.Errorf("esperaba rating 9, got: %d", p.Rating)
	}
	if !p.Active {
		t.Error("jugador nuevo debe estar activo")
	}
	if p.ID.String() == "" {
		t.Error("ID no debe estar vacío")
	}
}

func TestNew_EmptyName(t *testing.T) {
	_, err := player.New("", player.PositionDelantero, 8)
	if err != player.ErrEmptyName {
		t.Errorf("esperaba ErrEmptyName, got: %v", err)
	}
}

func TestNew_InvalidRatingTooLow(t *testing.T) {
	_, err := player.New("Vallejo", player.PositionDelantero, 0)
	if err != player.ErrInvalidRating {
		t.Errorf("esperaba ErrInvalidRating, got: %v", err)
	}
}

func TestNew_InvalidRatingTooHigh(t *testing.T) {
	_, err := player.New("Vallejo", player.PositionDelantero, 11)
	if err != player.ErrInvalidRating {
		t.Errorf("esperaba ErrInvalidRating, got: %v", err)
	}
}

func TestNew_RatingBoundaries(t *testing.T) {
	cases := []struct {
		rating int8
		valid  bool
	}{
		{1, true},
		{10, true},
		{0, false},
		{11, false},
		{5, true},
	}

	for _, tc := range cases {
		_, err := player.New("Test", player.PositionCreador, tc.rating)
		isValid := err == nil
		if isValid != tc.valid {
			t.Errorf("rating=%d: esperaba valid=%v, got=%v (err=%v)", tc.rating, tc.valid, isValid, err)
		}
	}
}

func TestNew_InvalidPosition(t *testing.T) {
	_, err := player.New("Test", player.Position("portero"), 7)
	if err != player.ErrInvalidPosition {
		t.Errorf("esperaba ErrInvalidPosition, got: %v", err)
	}
}

func TestNew_AllValidPositions(t *testing.T) {
	positions := []player.Position{
		player.PositionDelantero,
		player.PositionDefensa,
		player.PositionCreador,
	}
	for _, pos := range positions {
		_, err := player.New("Test", pos, 7)
		if err != nil {
			t.Errorf("posicion %s debe ser válida, got: %v", pos, err)
		}
	}
}

// --- Tests de UpdateRating ---

func TestUpdateRating_Valid(t *testing.T) {
	p, _ := player.New("Yuber", player.PositionDefensa, 9)

	err := p.UpdateRating(7)
	if err != nil {
		t.Fatalf("esperaba nil error, got: %v", err)
	}
	if p.Rating != 7 {
		t.Errorf("esperaba rating 7, got: %d", p.Rating)
	}
}

func TestUpdateRating_Invalid(t *testing.T) {
	p, _ := player.New("Yuber", player.PositionDefensa, 9)

	err := p.UpdateRating(0)
	if err != player.ErrInvalidRating {
		t.Errorf("esperaba ErrInvalidRating, got: %v", err)
	}
	// Rating no debe cambiar si hay error
	if p.Rating != 9 {
		t.Errorf("rating no debe cambiar con error, esperaba 9, got: %d", p.Rating)
	}
}

// --- Tests de Deactivate/Activate ---

func TestDeactivate(t *testing.T) {
	p, _ := player.New("Mickey", player.PositionDefensa, 6)
	p.Deactivate()

	if p.Active {
		t.Error("jugador debe estar inactivo después de Deactivate")
	}
}

func TestActivate(t *testing.T) {
	p, _ := player.New("Mickey", player.PositionDefensa, 6)
	p.Deactivate()
	p.Activate()

	if !p.Active {
		t.Error("jugador debe estar activo después de Activate")
	}
}

// --- Tests de CanPlay ---

func TestCanPlay_PrimaryPosition(t *testing.T) {
	p, _ := player.New("Yuber", player.PositionDefensa, 9)

	if !p.CanPlay(player.PositionDefensa) {
		t.Error("jugador debe poder jugar en su posición principal")
	}
}

func TestCanPlay_SecondaryPosition(t *testing.T) {
	p, _ := player.New("Tancho", player.PositionDefensa, 7)
	p.CanPlayPositions = []player.Position{player.PositionDelantero, player.PositionCreador}

	if !p.CanPlay(player.PositionDelantero) {
		t.Error("jugador debe poder jugar en posición secundaria")
	}
	if !p.CanPlay(player.PositionCreador) {
		t.Error("jugador debe poder jugar en posición secundaria")
	}
}

func TestCanPlay_UnavailablePosition(t *testing.T) {
	p, _ := player.New("Sebas", player.PositionDefensa, 8)
	// Sin posiciones secundarias

	if p.CanPlay(player.PositionDelantero) {
		t.Error("jugador no debe poder jugar en posición no asignada")
	}
}
