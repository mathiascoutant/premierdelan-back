package database

// Opérateurs d'agrégation MongoDB (évite les littéraux dupliqués)
const (
	BSONLookup  = "$lookup"
	BSONUnwind  = "$unwind"
	BSONMatch   = "$match"
	BSONRegex   = "$regex"
	BSONOptions = "$options"
)
