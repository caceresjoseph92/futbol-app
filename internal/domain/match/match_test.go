package match_test

import (
	"fmt"
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

// =============================================================================
// NUEVOS TESTS: Table-driven, distribución posicional, aleatorización
// =============================================================================

// --- Table-driven tests ------------------------------------------------------
// En lugar de un test por caso, definimos una tabla (slice de structs)
// con todos los escenarios. Go ejecuta cada fila como un sub-test con t.Run().
// Es el estándar en la comunidad Go — más legible y fácil de extender.

func TestFinish_TableDriven(t *testing.T) {
	cases := []struct {
		name    string
		score1  int
		score2  int
		wantErr error
	}{
		{"victoria equipo1",   3, 1, nil},
		{"victoria equipo2",   0, 2, nil},
		{"empate",             1, 1, nil},
		{"marcador negativo1", -1, 2, match.ErrInvalidScore},
		{"marcador negativo2", 2, -1, match.ErrInvalidScore},
		{"ambos negativos",    -1, -1, match.ErrInvalidScore},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newMatch()
			err := m.Finish(tc.score1, tc.score2)
			if err != tc.wantErr {
				t.Errorf("Finish(%d, %d) = %v, quería %v", tc.score1, tc.score2, err, tc.wantErr)
			}
		})
	}
}

// --- Distribución posicional -------------------------------------------------
// Verificamos que el algoritmo respeta la formación:
// con ≥2 creadores → cada equipo recibe ~1 creador
// con <2 creadores → los creadores pasan al pool de delanteros

func TestAssignTeams_PositionDistribution(t *testing.T) {
	m := newMatch()
	add12Players(m) // 6D + 2C + 4Del
	m.AssignTeams()

	defTeam1, defTeam2 := 0, 0
	for _, mp := range m.Players {
		if mp.PrimaryPosition == player.PositionDefensa {
			if mp.Team == 1 {
				defTeam1++
			} else {
				defTeam2++
			}
		}
	}
	// Cada equipo debe tener ~3 defensas (±1 por overflow)
	if defTeam1 < 2 || defTeam1 > 4 {
		t.Errorf("equipo 1 tiene %d defensas, esperaba ~3", defTeam1)
	}
	if defTeam2 < 2 || defTeam2 > 4 {
		t.Errorf("equipo 2 tiene %d defensas, esperaba ~3", defTeam2)
	}
}

func TestAssignTeams_FormacionSinCreadores(t *testing.T) {
	// Con <2 creadores, se tratan como delanteros → formación 3D-3F
	m := newMatch()
	jugadores := []*player.Player{
		newPlayer("D1", player.PositionDefensa,   8),
		newPlayer("D2", player.PositionDefensa,   8),
		newPlayer("D3", player.PositionDefensa,   7),
		newPlayer("D4", player.PositionDefensa,   7),
		newPlayer("D5", player.PositionDefensa,   7),
		newPlayer("D6", player.PositionDefensa,   6),
		newPlayer("C1", player.PositionCreador,   9), // solo 1 → pasa a delantero
		newPlayer("F1", player.PositionDelantero, 9),
		newPlayer("F2", player.PositionDelantero, 8),
		newPlayer("F3", player.PositionDelantero, 8),
		newPlayer("F4", player.PositionDelantero, 7),
		newPlayer("F5", player.PositionDelantero, 6),
	}
	for _, p := range jugadores {
		m.AddPlayer(p)
	}

	err := m.AssignTeams()
	if err != nil {
		t.Fatalf("AssignTeams falló: %v", err)
	}
	if len(m.Team1()) != 6 || len(m.Team2()) != 6 {
		t.Errorf("formación sin creadores debe producir 6v6")
	}
}

// --- Aleatorización ----------------------------------------------------------
// Ejecutamos AssignTeams 20 veces con jugadores de rating igual.
// Con shuffle aleatorio, en 20 intentos deben aparecer al menos 2 distribuciones distintas.

func TestAssignTeams_Randomizacion(t *testing.T) {
	seen := map[string]bool{}

	for i := 0; i < 20; i++ {
		m := newMatch()
		jugadores := []*player.Player{
			newPlayer("A", player.PositionDefensa,   8),
			newPlayer("B", player.PositionDefensa,   8),
			newPlayer("C", player.PositionDefensa,   8),
			newPlayer("D", player.PositionDefensa,   8),
			newPlayer("E", player.PositionDefensa,   8),
			newPlayer("F", player.PositionDefensa,   8),
			newPlayer("G", player.PositionDelantero, 8),
			newPlayer("H", player.PositionDelantero, 8),
			newPlayer("I", player.PositionDelantero, 8),
			newPlayer("J", player.PositionDelantero, 8),
			newPlayer("K", player.PositionDelantero, 8),
			newPlayer("L", player.PositionDelantero, 8),
		}
		for _, p := range jugadores {
			m.AddPlayer(p)
		}
		m.AssignTeams()

		// "firma" del equipo 1: nombres en orden
		firma := ""
		for _, mp := range m.Team1() {
			firma += mp.PlayerName
		}
		seen[firma] = true
	}

	if len(seen) < 2 {
		t.Errorf("AssignTeams no produce variedad en 20 ejecuciones (solo %d distribución distinta)", len(seen))
	}
}

// --- Balance de ratings ------------------------------------------------------
// Corremos el algoritmo muchas veces con jugadores variados y verificamos que
// la diferencia de ratings NUNCA supera un umbral razonable.

func TestAssignTeams_BalanceConsistente(t *testing.T) {
	for i := 0; i < 50; i++ {
		m := newMatch()
		add12Players(m)
		m.AssignTeams()

		diff := m.Team1Rating() - m.Team2Rating()
		if diff < 0 {
			diff = -diff
		}
		if diff > 6 {
			t.Errorf("iteración %d: diferencia de ratings muy alta: %d (E1=%d, E2=%d)",
				i, diff, m.Team1Rating(), m.Team2Rating())
		}
	}
}

// Silenciar el "imported and not used" de uuid si no se usa en nuevos tests
var _ = uuid.UUID{}
var _ = fmt.Sprintf

// =============================================================================
// BENCHMARKS
// =============================================================================
//
// Los benchmarks miden el rendimiento real del código.
// El testing framework los ejecuta con `go test -bench=.`
//
// ¿Cómo funcionan?
//   - b.N: el framework ajusta N automáticamente hasta que el resultado sea estable.
//     En las primeras corridas puede ser 100, luego 10000, etc.
//   - b.ResetTimer(): reinicia el contador después de setup costoso (crear jugadores).
//     Sin esto, el tiempo de setup contamina el resultado del benchmark.
//   - b.RunParallel: lanza el benchmark en múltiples goroutines simultáneas,
//     útil para detectar condiciones de carrera o medir throughput concurrente.
//
// Comandos útiles:
//   go test ./internal/domain/match/... -bench=. -benchmem
//   go test ./internal/domain/match/... -bench=BenchmarkAssignTeams -count=5
//
// -benchmem muestra:
//   - allocs/op: allocaciones de heap por operación
//   - B/op: bytes alocados por operación

// BenchmarkAssignTeams mide el tiempo de ejecución del algoritmo de balanceo.
func BenchmarkAssignTeams(b *testing.B) {
	// Setup: preparar jugadores fuera del loop de medición
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Necesitamos un match limpio en cada iteración (AssignTeams modifica el estado)
		b.StopTimer()
		m := newMatch()
		add12Players(m)
		b.StartTimer()

		m.AssignTeams()
	}
}

// BenchmarkAssignTeams_Parallel mide el throughput con múltiples goroutines.
// Cada goroutine corre el benchmark de forma independiente.
func BenchmarkAssignTeams_Parallel(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m := newMatch()
			add12Players(m)
			m.AssignTeams()
		}
	})
}
