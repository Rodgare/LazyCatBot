package models

// DigestPlayer is one row in a weekly digest top list.
type DigestPlayer struct {
	Name          string
	ClassID       int
	SpecID        int
	Value         int
	BlackDiamonds int
	Title         string
}

// FirstKill is a boss killed for the first time inside the digest window.
type FirstKill struct {
	BossID   int
	BossName string
	KilledAt int64
}

// AttendanceEntry is one player's attendance inside a window.
type AttendanceEntry struct {
	Name          string
	ClassID       int
	SpecID        int
	Role          string
	Kills         int
	LastSeen      int64
	Ilvl          int
	SetPieces     string
	BlackDiamonds int
	Title         string
}

// WeeklyDigest is everything the weekly image needs, assembled by storage.
type WeeklyDigest struct {
	GuildName string
	Realm     string
	Since     int64
	Until     int64

	TotalKills int
	FirstKills []FirstKill
	TopDps     []DigestPlayer
	TopHps     []DigestPlayer
	Attendance []AttendanceEntry
	MythicTop  []GuildMythicPlayer
}

// PlayerKillProfile is the kill-history part of a player card. Character data
// is joined in by the handler from the caches.
type PlayerKillProfile struct {
	Name      string
	GUID      int
	ClassID   int
	SpecID    int
	Role      string
	Ilvl      int
	T4        int
	SetPieces string

	TotalKills  int
	AttendedPct int
	BestDps     int
	BestHps     int
	FirstSeen   int64
	LastSeen    int64
}
