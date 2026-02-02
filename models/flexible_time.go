package models

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/bsontype"
)

// FlexibleTime gère plusieurs formats de dates
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON implémente le unmarshaler pour accepter plusieurs formats de dates
func (ft *FlexibleTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "null" || s == "" {
		ft.Time = time.Time{}
		return nil
	}

	// Charger la timezone de Paris
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		paris = time.FixedZone("CET", 2*3600) // Fallback: UTC+2
	}

	// TOUS LES FORMATS sont parsés en heure française
	// Peu importe ce que le frontend envoie, on garde l'heure telle quelle
	formats := []string{
		"2006-01-02T15:04:05", // "2025-12-31T20:00:00"
		"2006-01-02T15:04",    // "2025-12-31T20:00"
		time.RFC3339,          // "2025-12-31T20:00:00Z" (on ignore le Z)
		time.RFC3339Nano,      // Avec nanosecondes
	}

	for _, layout := range formats {
		// TOUJOURS parser en timezone France
		parsedTime, parseErr := time.ParseInLocation(layout, s, paris)
		if parseErr == nil {
			ft.Time = parsedTime
			return nil
		}
	}

	return fmt.Errorf("format de date invalide: %s", s)
}

// MarshalJSON retourne la date EN HEURE FRANÇAISE (même si MongoDB stocke en UTC)
func (ft FlexibleTime) MarshalJSON() ([]byte, error) {
	if ft.Time.IsZero() {
		return []byte("null"), nil
	}

	// MongoDB stocke toujours en UTC, donc on doit reconvertir en heure française
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		paris = time.FixedZone("CET", 2*3600)
	}

	// Convertir en timezone France
	frenchTime := ft.Time.In(paris)

	// Retourner SANS le Z (format simple : YYYY-MM-DDTHH:MM:SS)
	return []byte("\"" + frenchTime.Format("2006-01-02T15:04:05") + "\""), nil
}

// MarshalBSONValue dit à MongoDB de stocker FlexibleTime comme une date (pas un document)
func (ft *FlexibleTime) MarshalBSONValue() (bsontype.Type, []byte, error) {
	if ft == nil || ft.Time.IsZero() {
		// Retourner null pour les dates vides
		return bsontype.Null, nil, nil
	}

	// Convertir en millisecondes depuis Unix epoch (format MongoDB)
	timestampMs := ft.Time.UnixMilli()

	// Créer le buffer pour stocker l'int64 en little-endian
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(timestampMs))

	return bsontype.DateTime, buf, nil
}

// UnmarshalBSONValue permet au driver MongoDB de décoder une date en FlexibleTime
func (ft *FlexibleTime) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	// Si c'est une date MongoDB, on la decode
	if t == bsontype.DateTime {
		// Vérifier qu'on a assez de bytes pour un timestamp (8 bytes = int64)
		if len(data) < 8 {
			return fmt.Errorf("invalid DateTime data: need 8 bytes, got %d", len(data))
		}

		// Lire le timestamp MongoDB (int64 en little-endian, en millisecondes depuis Unix epoch)
		timestampMs := int64(binary.LittleEndian.Uint64(data[:8]))

		// Convertir millisecondes en secondes et nanosecondes
		seconds := timestampMs / 1000
		nanos := (timestampMs % 1000) * 1000000

		// Créer le time.Time
		ft.Time = time.Unix(seconds, nanos)
		return nil
	}

	// Si c'est null, on retourne une date vide
	if t == bsontype.Null {
		ft.Time = time.Time{}
		return nil
	}

	// Pour tout autre type, on essaie de décoder comme un time.Time standard
	// Cela peut arriver si l'événement a été créé d'une autre manière
	return fmt.Errorf("cannot decode %v into FlexibleTime (expected DateTime, got %v)", t, t)
}
