package main

import (
	"fmt"
	"log"
	"premier-an-backend/utils"
)

func main() {
	log.Println("🔐 Génération des clés VAPID...")
	
	publicKey, privateKey, err := utils.GenerateVAPIDKeys()
	if err != nil {
		log.Fatalf("❌ Erreur lors de la génération des clés: %v", err)
	}

	fmt.Println("\n✅ Clés VAPID générées avec succès!")
	fmt.Println("\nAjoutez ces lignes dans votre fichier .env:\n")
	fmt.Println("VAPID_PUBLIC_KEY=" + publicKey)
	fmt.Println("VAPID_PRIVATE_KEY=" + privateKey)
	fmt.Println("VAPID_SUBJECT=mailto:votre-email@example.com")
	fmt.Println("\n⚠️  Important: Ne partagez JAMAIS votre clé privée!")
}

