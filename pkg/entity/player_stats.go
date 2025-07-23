package entity

type DefaultStat struct {
	Percent    float64
	LastUpdate int64
}
type PlayerStat struct {
	CP     DefaultStat
	HP     DefaultStat
	MP     DefaultStat
	Target struct {
		HpPercent  float64
		LastUpdate int64
	}
}
type StatStr struct {
	Player map[uint32]PlayerStat
	Party  map[uint8]struct {
		HP DefaultStat
	}
}
