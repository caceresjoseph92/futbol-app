package player_test

import (
	"context"
	"testing"

	appplayer "futbol-app/internal/application/player"
	"futbol-app/internal/domain/player"

	"github.com/google/uuid"
)

// --- Mock del repositorio ---
// Implementa player.Repository en memoria para testing.
// No necesitamos base de datos real — esto es la ventaja de la Clean Architecture.

type mockPlayerRepo struct {
	players map[uuid.UUID]*player.Player
}

func newMockRepo() *mockPlayerRepo {
	return &mockPlayerRepo{players: make(map[uuid.UUID]*player.Player)}
}

func (m *mockPlayerRepo) Save(ctx context.Context, p *player.Player) error {
	m.players[p.ID] = p
	return nil
}

func (m *mockPlayerRepo) FindByID(ctx context.Context, id uuid.UUID) (*player.Player, error) {
	p, ok := m.players[id]
	if !ok {
		return nil, player.ErrPlayerNotFound
	}
	return p, nil
}

func (m *mockPlayerRepo) FindAll(ctx context.Context) ([]*player.Player, error) {
	var result []*player.Player
	for _, p := range m.players {
		if p.Active {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPlayerRepo) FindAllIncludingInactive(ctx context.Context) ([]*player.Player, error) {
	var result []*player.Player
	for _, p := range m.players {
		result = append(result, p)
	}
	return result, nil
}

func (m *mockPlayerRepo) Update(ctx context.Context, p *player.Player) error {
	m.players[p.ID] = p
	return nil
}

func (m *mockPlayerRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.players, id)
	return nil
}

// --- Tests del servicio ---

func TestCreatePlayer_Success(t *testing.T) {
	svc := appplayer.NewService(newMockRepo())

	p, err := svc.CreatePlayer(context.Background(), appplayer.CreatePlayerInput{
		Name:            "Yuber",
		PrimaryPosition: player.PositionDefensa,
		Rating:          9,
	})

	if err != nil {
		t.Fatalf("esperaba nil, got: %v", err)
	}
	if p.Name != "Yuber" {
		t.Errorf("nombre incorrecto: %s", p.Name)
	}
	if p.ID == uuid.Nil {
		t.Error("ID no debe ser nil")
	}
}

func TestCreatePlayer_InvalidRating(t *testing.T) {
	svc := appplayer.NewService(newMockRepo())

	_, err := svc.CreatePlayer(context.Background(), appplayer.CreatePlayerInput{
		Name:            "Test",
		PrimaryPosition: player.PositionDefensa,
		Rating:          0, // inválido
	})

	if err == nil {
		t.Error("esperaba error por rating inválido")
	}
}

func TestGetPlayer_Found(t *testing.T) {
	repo := newMockRepo()
	svc := appplayer.NewService(repo)

	created, _ := svc.CreatePlayer(context.Background(), appplayer.CreatePlayerInput{
		Name:            "Vallejo",
		PrimaryPosition: player.PositionDelantero,
		Rating:          9,
	})

	found, err := svc.GetPlayer(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("esperaba nil, got: %v", err)
	}
	if found.Name != "Vallejo" {
		t.Errorf("nombre incorrecto: %s", found.Name)
	}
}

func TestGetPlayer_NotFound(t *testing.T) {
	svc := appplayer.NewService(newMockRepo())

	_, err := svc.GetPlayer(context.Background(), uuid.New())
	if err != player.ErrPlayerNotFound {
		t.Errorf("esperaba ErrPlayerNotFound, got: %v", err)
	}
}

func TestUpdateRating_Success(t *testing.T) {
	svc := appplayer.NewService(newMockRepo())

	p, _ := svc.CreatePlayer(context.Background(), appplayer.CreatePlayerInput{
		Name:            "Yuber",
		PrimaryPosition: player.PositionDefensa,
		Rating:          9,
	})

	updated, err := svc.UpdateRating(context.Background(), p.ID, 7)
	if err != nil {
		t.Fatalf("esperaba nil, got: %v", err)
	}
	if updated.Rating != 7 {
		t.Errorf("esperaba rating 7, got: %d", updated.Rating)
	}
}

func TestDeactivatePlayer(t *testing.T) {
	svc := appplayer.NewService(newMockRepo())

	p, _ := svc.CreatePlayer(context.Background(), appplayer.CreatePlayerInput{
		Name:            "Mickey",
		PrimaryPosition: player.PositionDefensa,
		Rating:          6,
	})

	if err := svc.DeactivatePlayer(context.Background(), p.ID); err != nil {
		t.Fatalf("esperaba nil, got: %v", err)
	}

	deactivated, _ := svc.GetPlayer(context.Background(), p.ID)
	if deactivated.Active {
		t.Error("jugador debe estar inactivo")
	}
}

func TestListPlayers_OnlyActive(t *testing.T) {
	svc := appplayer.NewService(newMockRepo())

	p1, _ := svc.CreatePlayer(context.Background(), appplayer.CreatePlayerInput{
		Name: "Activo", PrimaryPosition: player.PositionDefensa, Rating: 7,
	})
	p2, _ := svc.CreatePlayer(context.Background(), appplayer.CreatePlayerInput{
		Name: "Inactivo", PrimaryPosition: player.PositionDefensa, Rating: 7,
	})
	svc.DeactivatePlayer(context.Background(), p2.ID)
	_ = p1

	players, _ := svc.ListPlayers(context.Background())
	for _, p := range players {
		if !p.Active {
			t.Errorf("ListPlayers no debe retornar jugadores inactivos: %s", p.Name)
		}
	}
}
