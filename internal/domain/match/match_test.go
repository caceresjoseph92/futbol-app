package match_test

import (
	"testing"
	"time"

	"futbol-app/internal/domain/match"
	"futbol-app/internal/domain/player"

	"github.com/google/uuid"
)

// --- helpers ---

func newPlayer(name string, pos player.Position, rating int8) *player.Player {
	p, _ := player.New(name, pos, rating)
	return p
}

func newMatch() *match.Match {
	return match.New(time.Now(), "Futguardian 1 y Futguardian 2", uuid.New())
}

func add12Players(m *match.Match) []*player.Player {
	players := []*player.Player{
		newPlayer("Yuber",   player.PositionDefensa,   9),
		newPlayer("Vallejo", player.PositionDelantero, 9),
		newPlayer("Hanson",  player.PositionDelantero, 8),
		newPlayer("Silva",   player.PositionCreador,   8),
		newPlayer("Florez",  player.PositionCreador,   9),
		newPlayer("Yefry",   player.PositionDelantero, 8),
		newPlayer("Ossa",    player.PositionDefensa,   7),
		newPlayer("Sebas",   player.PositionDefensa,   8),
		newPlayer("Mickey",  player.PositionDefensa,   6),
		newPlayer("Joseph",  player.PositionDefensa,   8),
		newPlayer("CarlosL", player.PositionDefensa,   7),
		newPlayer("Tancho",  player.PositionDefensa,   7),
	}
	for _, p := range players {
		m.AddPlayer(p)
	}
	return players
}

// --- Tests de creación ---

func TestNew_InitialStatus(t *testing.T) {
	m := newMatch()
	if m.Status != match.StatusDraft {
		t.Errorf("nuevo partido debe estar en draft, got: %s", m.Status)
	}
	if m.ID.String() == "" {
		t.Error("ID no debe estar vacío")
	}
}

// --- Tests de AddPlayer ---

func TestAddPlayer_Success(t *testing.T) {
	m := newMatch()
	p := newPlayer("Yuber", player.PositionDefensa, 9)

	err := m.AddPlayer(p)
	if err != nil {
		t.Fatalf("esperaba nil, got: %v", err)
	}
	if len(m.Players) != 1 {
		t.Errorf("esperaba 1 jugador, got: %d", len(m.Players))
	}
}

func TestAddPlayer_Duplicate(t *testing.T) {
	m := newMatch()
	p := newPlayer("Yuber", player.PositionDefensa, 9)
	m.AddPlayer(p)

	err := m.AddPlayer(p)
	if err != match.ErrPlayerAlreadyInMatch {
		t.Errorf("esperaba ErrPlayerAlreadyInMatch, got: %v", err)
	}
}

func TestAddPlayer_RatingSnapshot(t *testing.T) {
	m := newMatch()
	p := newPlayer("Yuber", player.PositionDefensa, 9)
	m.AddPlayer(p)

	// Cambiar rating después de agregar al partido
	p.UpdateRating(5)

	// El snapshot debe conservar el rating original
	if m.Players[0].RatingSnapshot != 9 {
		t.Errorf("snapshot debe ser 9 (rating al momento), got: %d", m.Players[0].RatingSnapshot)
	}
}

// --- Tests de AssignTeams ---

func TestAssignTeams_RequiresExactly12(t *testing.T) {
	m := newMatch()
	// Solo 5 jugadores
	for i := 0; i < 5; i++ {
		m.AddPlayer(newPlayer("P", player.PositionDefensa, 7))
	}
	err := m.AssignTeams()
	if err != match.ErrInvalidPlayerCount {
		t.Errorf("esperaba ErrInvalidPlayerCount, got: %v", err)
	}
}

func TestAssignTeams_BalancedTeams(t *testing.T) {
	m := newMatch()
	add12Players(m)

	err := m.AssignTeams()
	if err != nil {
		t.Fatalf("esperaba nil, got: %v", err)
	}

	team1 := m.Team1()
	team2 := m.Team2()

	if len(team1) != 6 {
		t.Errorf("equipo 1 debe tener 6 jugadores, got: %d", len(team1))
	}
	if len(team2) != 6 {
		t.Errorf("equipo 2 debe tener 6 jugadores, got: %d", len(team2))
	}

	// La diferencia de ratings no debe ser mayor a 5 (razonablemente balanceado)
	diff := m.Team1Rating() - m.Team2Rating()
	if diff < 0 {
		diff = -diff
	}
	if diff > 5 {
		t.Errorf("diferencia de ratings muy alta: %d (equipo1=%d, equipo2=%d)",
			diff, m.Team1Rating(), m.Team2Rating())
	}
}

func TestAssignTeams_AllPlayersAssigned(t *testing.T) {
	m := newMatch()
	add12Players(m)
	m.AssignTeams()

	for _, mp := range m.Players {
		if mp.Team == 0 {
			t.Errorf("jugador %s no tiene equipo asignado", mp.PlayerName)
		}
	}
}

// --- Tests de TransferPlayer ---

func TestTransferPlayer_Success(t *testing.T) {
	m := newMatch()
	players := add12Players(m)
	m.AssignTeams()

	// Encontrar un jugador del equipo 1
	var targetID uuid.UUID
	for _, mp := range m.Team1() {
		targetID = mp.PlayerID
		break
	}

	err := m.TransferPlayer(targetID)
	if err != nil {
		t.Fatalf("esperaba nil, got: %v", err)
	}
	_ = players // usado indirectamente

	// Verificar que cambió de equipo
	for _, mp := range m.Players {
		if mp.PlayerID == targetID && mp.Team != 2 {
			t.Errorf("jugador debe estar en equipo 2 después de transferir")
		}
	}
}

func TestTransferPlayer_NotFound(t *testing.T) {
	m := newMatch()
	err := m.TransferPlayer(uuid.New())
	if err != match.ErrPlayerNotInMatch {
		t.Errorf("esperaba ErrPlayerNotInMatch, got: %v", err)
	}
}

// --- Tests de SwapPlayers ---

func TestSwapPlayers_Success(t *testing.T) {
	m := newMatch()
	add12Players(m)
	m.AssignTeams()

	team1Before := m.Team1()
	team2Before := m.Team2()

	p1 := team1Before[0].PlayerID // equipo 1
	p2 := team2Before[0].PlayerID // equipo 2

	err := m.SwapPlayers(p1, p2)
	if err != nil {
		t.Fatalf("esperaba nil, got: %v", err)
	}

	// p1 debe estar en equipo 2 y p2 en equipo 1
	for _, mp := range m.Players {
		if mp.PlayerID == p1 && mp.Team != 2 {
			t.Errorf("p1 debe estar en equipo 2 después del swap")
		}
		if mp.PlayerID == p2 && mp.Team != 1 {
			t.Errorf("p2 debe estar en equipo 1 después del swap")
		}
	}
}

// --- Tests de Publish ---

func TestPublish_WithAssignedTeams(t *testing.T) {
	m := newMatch()
	add12Players(m)
	m.AssignTeams()

	err := m.Publish()
	if err != nil {
		t.Fatalf("esperaba nil, got: %v", err)
	}
	if m.Status != match.StatusPublished {
		t.Errorf("esperaba status published, got: %s", m.Status)
	}
}

func TestPublish_WithoutTeams(t *testing.T) {
	m := newMatch()
	add12Players(m)
	// Sin AssignTeams — players tienen team=0

	err := m.Publish()
	if err == nil {
		t.Error("no se debe poder publicar sin equipos asignados")
	}
}

func TestPublish_AlreadyPublished(t *testing.T) {
	m := newMatch()
	add12Players(m)
	m.AssignTeams()
	m.Publish()

	err := m.Publish()
	if err != match.ErrMatchAlreadyPublished {
		t.Errorf("esperaba ErrMatchAlreadyPublished, got: %v", err)
	}
}

// --- Tests de Finish ---

func TestFinish_ValidScore(t *testing.T) {
	m := newMatch()
	err := m.Finish(3, 1)

	if err != nil {
		t.Fatalf("esperaba nil, got: %v", err)
	}
	if m.Status != match.StatusFinished {
		t.Errorf("esperaba status finished, got: %s", m.Status)
	}
	if *m.Team1Score != 3 || *m.Team2Score != 1 {
		t.Errorf("marcador incorrecto: %d-%d", *m.Team1Score, *m.Team2Score)
	}
}

func TestFinish_NegativeScore(t *testing.T) {
	m := newMatch()
	err := m.Finish(-1, 2)
	if err != match.ErrInvalidScore {
		t.Errorf("esperaba ErrInvalidScore, got: %v", err)
	}
}

func TestFinish_ZeroScore(t *testing.T) {
	m := newMatch()
	err := m.Finish(0, 0)
	if err != nil {
		t.Errorf("empate 0-0 debe ser válido, got: %v", err)
	}
}
