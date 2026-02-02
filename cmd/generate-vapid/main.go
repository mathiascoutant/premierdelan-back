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

	fmt.Println("✅ Clés VAPID générées avec succès!")
	fmt.Println()
	fmt.Println("Ajoutez ces lignes dans votre fichier .env:")
	fmt.Println()
	fmt.Println("VAPID_PUBLIC_KEY=" + publicKey)
	fmt.Println("VAPID_PRIVATE_KEY=" + privateKey)
	fmt.Println("VAPID_SUBJECT=mailto:votre-email@example.com")
	fmt.Println()
	fmt.Println("⚠️  Important: Ne partagez JAMAIS votre clé privée!")
}

