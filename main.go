package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"premier-an-backend/config"
	"premier-an-backend/database"
	"premier-an-backend/handlers"
	"premier-an-backend/middleware"
	"premier-an-backend/services"
	"premier-an-backend/utils"
	"premier-an-backend/websocket"
	"syscall"

	"github.com/gorilla/mux"
)

func main() {
	// Charger la configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Erreur lors du chargement de la configuration: %v", err)
	}

	// Connexion à MongoDB
	if err := database.Connect(cfg.MongoURI, cfg.MongoDB); err != nil {
		log.Fatalf("❌ Erreur de connexion à MongoDB: %v", err)
	}
	defer database.Close()

	// Initialiser Firebase Cloud Messaging (optionnel pour Railway)
	fcmService, err := services.NewFCMService(cfg.FirebaseCredentialsFile)
	if err != nil {
		log.Printf("⚠️  Erreur d'initialisation Firebase: %v", err)
		log.Println("⚠️  Le serveur démarre SANS notifications push")
		log.Println("💡 Pour activer Firebase : configurez FIREBASE_CREDENTIALS_BASE64 dans Railway")
		fcmService = services.NewDisabledFCMService()
	} else {
		log.Println("✓ Firebase Cloud Messaging initialisé")
		
		// Initialiser et démarrer le cron job pour les notifications automatiques
		notificationCron := services.NewNotificationCron(database.DB, fcmService)
		notificationCron.Start()
	}

	// Créer le routeur
	router := mux.NewRouter()
	
	// Créer un routeur sans middleware pour WebSocket
	rawRouter := mux.NewRouter()

	// Appliquer les middlewares globaux (SAUF pour WebSocket)
	router.Use(middleware.Logging)
	router.Use(middleware.CORS(cfg.CORSOrigins))

	// Créer les repositories
	siteSettingRepo := database.NewSiteSettingRepository(database.DB)
	userRepo := database.NewUserRepository(database.DB)
	chatRepo := database.NewChatRepository(database.DB)
	fcmTokenRepo := database.NewFCMTokenRepository(database.DB)

	// Créer les handlers
	authHandler := handlers.NewAuthHandler(database.DB, cfg.JWTSecret, fcmService)
	notificationHandler := handlers.NewNotificationHandler(
		database.DB,
		cfg.VAPIDPublicKey,
		cfg.VAPIDPrivateKey,
		cfg.VAPIDSubject,
	)
	fcmHandler := handlers.NewFCMHandler(database.DB, fcmService)
	adminHandler := handlers.NewAdminHandler(database.DB, fcmService)
	eventHandler := handlers.NewEventHandler(database.DB)
	inscriptionHandler := handlers.NewInscriptionHandler(database.DB, fcmService)
	mediaHandler := handlers.NewMediaHandler(database.DB)
	alertHandler := handlers.NewAlertHandler(database.DB, fcmService)
	themeHandler := handlers.NewThemeHandler(siteSettingRepo, userRepo)
	
	// Initialiser le hub WebSocket pour le chat (avec repositories pour la présence)
	wsHub := websocket.NewHub(userRepo, chatRepo)
	go wsHub.Run()
	log.Println("✅ Hub WebSocket initialisé et en cours d'exécution")
	
	chatHandler := handlers.NewChatHandler(chatRepo, userRepo, fcmTokenRepo, fcmService, wsHub)
	testNotifHandler := handlers.NewTestNotifHandler(fcmTokenRepo, fcmService)
	wsHandler := websocket.NewHandler(wsHub, cfg.JWTSecret)
	chatGroupHandler := handlers.NewChatGroupHandler(database.DB, fcmService, wsHub)

	// Middleware Guest pour empêcher l'accès si déjà connecté
	guestMiddleware := middleware.Guest(cfg.JWTSecret)

	// Routes publiques - Compatible avec votre front
	// Ces routes sont protégées par le middleware Guest (refusent les utilisateurs déjà connectés)
	router.Handle("/api/inscription", guestMiddleware(http.HandlerFunc(authHandler.Register))).Methods("POST", "OPTIONS")
	router.Handle("/api/connexion", guestMiddleware(http.HandlerFunc(authHandler.Login))).Methods("POST", "OPTIONS")
	
	// Routes alternatives (pour compatibilité)
	router.Handle("/api/auth/register", guestMiddleware(http.HandlerFunc(authHandler.Register))).Methods("POST", "OPTIONS")
	router.Handle("/api/auth/login", guestMiddleware(http.HandlerFunc(authHandler.Login))).Methods("POST", "OPTIONS")

	// Route de santé (health check)
	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		utils.RespondSuccess(w, "Le serveur fonctionne correctement", map[string]string{
			"status":   "ok",
			"env":      cfg.Environment,
			"database": "MongoDB",
		})
	}).Methods("GET")

	// Routes publiques des événements
	router.HandleFunc("/api/evenements/public", eventHandler.GetPublicEvents).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/evenements/{event_id}", eventHandler.GetPublicEvent).Methods("GET", "OPTIONS")

	// Routes publiques des médias (galerie)
	router.HandleFunc("/api/evenements/{event_id}/medias", mediaHandler.GetMedias).Methods("GET", "OPTIONS")

	// Route d'alertes critiques (publique - pas d'auth pour permettre les alertes en cas d'erreur)
	router.HandleFunc("/api/alerts/critical", alertHandler.SendCriticalAlert).Methods("POST", "OPTIONS")

	// Route thème global (publique)
	router.HandleFunc("/api/theme", themeHandler.GetGlobalTheme).Methods("GET", "OPTIONS")

	// Routes de notifications (publiques)
	router.HandleFunc("/api/notifications/vapid-public-key", notificationHandler.GetVAPIDPublicKey).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/notifications/subscribe", notificationHandler.Subscribe).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/notifications/unsubscribe", notificationHandler.Unsubscribe).Methods("POST", "OPTIONS")
	
	// Routes FCM (Firebase Cloud Messaging) - Publiques
	router.HandleFunc("/api/fcm/vapid-key", func(w http.ResponseWriter, r *http.Request) {
		utils.RespondJSON(w, http.StatusOK, map[string]string{
			"vapidKey": cfg.FCMVAPIDKey,
		})
	}).Methods("GET", "OPTIONS")
	router.HandleFunc("/api/fcm/subscribe", fcmHandler.Subscribe).Methods("POST", "OPTIONS")

	// Routes protégées
	protected := router.PathPrefix("/api").Subrouter()
	protected.Use(middleware.Auth(cfg.JWTSecret))
	
	// Routes de notifications (VAPID - ancienne méthode, garde pour compatibilité)
	protected.HandleFunc("/notification/test", notificationHandler.SendTestNotification).Methods("POST", "OPTIONS")
	
	// Routes FCM (Firebase Cloud Messaging) - RECOMMANDÉ
	protected.HandleFunc("/fcm/send", fcmHandler.SendNotification).Methods("POST", "OPTIONS")
	protected.HandleFunc("/fcm/send-to-user", fcmHandler.SendToUser).Methods("POST", "OPTIONS")
	
	// 🧪 ROUTES DE TEST ULTRA SIMPLE
	protected.HandleFunc("/test/simple-notif", testNotifHandler.SendSimpleTest).Methods("POST", "OPTIONS")
	protected.HandleFunc("/test/list-tokens", testNotifHandler.ListMyTokens).Methods("POST", "OPTIONS")
	
	// 🔌 ROUTE WEBSOCKET CHAT (SANS middleware - Render.com supporté !)
	// La route WebSocket doit être sur rawRouter pour éviter le wrapping du ResponseWriter
	rawRouter.HandleFunc("/ws/chat", wsHandler.ServeWS).Methods("GET")
	
	// Routes Admin (protégées par Auth + RequireAdmin)
	adminRouter := protected.PathPrefix("/admin").Subrouter()
	adminRouter.Use(middleware.RequireAdmin(database.DB))
	
	// Gestion des utilisateurs
	adminRouter.HandleFunc("/utilisateurs", adminHandler.GetUsers).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/utilisateurs/{id}", adminHandler.UpdateUser).Methods("PUT", "OPTIONS")
	adminRouter.HandleFunc("/utilisateurs/{id}", adminHandler.DeleteUser).Methods("DELETE", "OPTIONS")
	
	// Gestion des événements
	adminRouter.HandleFunc("/evenements", adminHandler.GetEvents).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/evenements", adminHandler.CreateEvent).Methods("POST", "OPTIONS")
	adminRouter.HandleFunc("/evenements/{event_id}", adminHandler.GetEvent).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/evenements/{id}", adminHandler.UpdateEvent).Methods("PUT", "OPTIONS")
	adminRouter.HandleFunc("/evenements/{id}", adminHandler.DeleteEvent).Methods("DELETE", "OPTIONS")
	
	// Statistiques
	adminRouter.HandleFunc("/stats", adminHandler.GetStats).Methods("GET", "OPTIONS")
	
	// Notifications admin
	adminRouter.HandleFunc("/notifications/send", adminHandler.SendAdminNotification).Methods("POST", "OPTIONS")
	
	// Codes soirée
	adminRouter.HandleFunc("/codes-soiree", adminHandler.GetAllCodesSoiree).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/code-soiree/generate", adminHandler.GenerateCodeSoiree).Methods("POST", "OPTIONS")
	adminRouter.HandleFunc("/code-soiree/current", adminHandler.GetCurrentCodeSoiree).Methods("GET", "OPTIONS")
	
	// Chat admin
	adminRouter.HandleFunc("/chat/conversations", chatHandler.GetConversations).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/chat/conversations/{id}/messages", chatHandler.GetMessages).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/chat/conversations/{id}/messages", chatHandler.SendMessage).Methods("POST", "OPTIONS")
	adminRouter.HandleFunc("/chat/conversations/{id}/mark-read", chatHandler.MarkConversationAsRead).Methods("POST", "OPTIONS")
	adminRouter.HandleFunc("/chat/admins/search", chatHandler.SearchAdmins).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/chat/invitations", chatHandler.SendInvitation).Methods("POST", "OPTIONS")
	adminRouter.HandleFunc("/chat/invitations", chatHandler.GetInvitations).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/chat/invitations/{id}/respond", chatHandler.RespondToInvitation).Methods("PUT", "OPTIONS")
	adminRouter.HandleFunc("/chat/notifications/send", chatHandler.SendChatNotification).Methods("POST", "OPTIONS")
	
	// 👥 Routes Groupes de chat (admin)
	adminRouter.HandleFunc("/chat/groups", chatGroupHandler.CreateGroup).Methods("POST", "OPTIONS")
	adminRouter.HandleFunc("/chat/groups", chatGroupHandler.GetGroups).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/chat/groups/{group_id}/invite", chatGroupHandler.InviteToGroup).Methods("POST", "OPTIONS")
	adminRouter.HandleFunc("/chat/groups/{group_id}/members", chatGroupHandler.GetGroupMembers).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/chat/groups/{group_id}/leave", chatGroupHandler.LeaveGroup).Methods("POST", "OPTIONS")
	adminRouter.HandleFunc("/chat/groups/{group_id}/pending-invitations", chatGroupHandler.GetGroupPendingInvitations).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/chat/groups/{group_id}/invitations/pending", chatGroupHandler.GetGroupPendingInvitations).Methods("GET", "OPTIONS") // Alias pour frontend
	adminRouter.HandleFunc("/chat/groups/{group_id}/messages", chatGroupHandler.SendMessage).Methods("POST", "OPTIONS")
	adminRouter.HandleFunc("/chat/groups/{group_id}/messages", chatGroupHandler.GetMessages).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/chat/groups/{group_id}/mark-read", chatGroupHandler.MarkAsRead).Methods("POST", "OPTIONS")
	adminRouter.HandleFunc("/chat/group-invitations/pending", chatGroupHandler.GetPendingInvitations).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/chat/group-invitations/{invitation_id}/respond", chatGroupHandler.RespondToInvitation).Methods("PUT", "OPTIONS")
	adminRouter.HandleFunc("/chat/group-invitations/{invitation_id}/cancel", chatGroupHandler.CancelInvitation).Methods("DELETE", "OPTIONS")
	adminRouter.HandleFunc("/chat/users/search", chatGroupHandler.SearchUsers).Methods("GET", "OPTIONS")
	
	// Route protégée exemple
	protected.HandleFunc("/protected/profile", func(w http.ResponseWriter, r *http.Request) {
		claims := middleware.GetUserFromContext(r.Context())
		if claims == nil {
			utils.RespondError(w, http.StatusUnauthorized, "Utilisateur non authentifié")
			return
		}
		
		utils.RespondSuccess(w, "Profil récupéré avec succès", map[string]interface{}{
			"user_id": claims.UserID,
			"email":   claims.Email,
		})
	}).Methods("GET")

	// Routes d'inscription aux événements (protégées - authentification requise)
	protected.HandleFunc("/evenements/{event_id}/inscription", inscriptionHandler.CreateInscription).Methods("POST", "OPTIONS")
	protected.HandleFunc("/evenements/{event_id}/inscription", inscriptionHandler.GetInscription).Methods("GET", "OPTIONS")
	protected.HandleFunc("/evenements/{event_id}/inscription", inscriptionHandler.UpdateInscription).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/evenements/{event_id}/desinscription", inscriptionHandler.DeleteInscription).Methods("DELETE", "OPTIONS")
	
	// Route pour récupérer les événements auxquels l'utilisateur est inscrit
	protected.HandleFunc("/mes-evenements", inscriptionHandler.GetMesEvenements).Methods("GET", "OPTIONS")

	// Route thème global (protégée - admin uniquement)
	protected.HandleFunc("/theme", themeHandler.SetGlobalTheme).Methods("POST", "OPTIONS")

	// Routes de chat (protégées - admin uniquement, pour compatibilité frontend)
	protected.HandleFunc("/chat/conversations", chatHandler.GetConversations).Methods("GET", "OPTIONS")
	protected.HandleFunc("/chat/conversations/{id}/messages", chatHandler.GetMessages).Methods("GET", "OPTIONS")
	protected.HandleFunc("/chat/conversations/{id}/messages", chatHandler.SendMessage).Methods("POST", "OPTIONS")
	protected.HandleFunc("/chat/conversations/{id}/mark-read", chatHandler.MarkConversationAsRead).Methods("POST", "OPTIONS")
	protected.HandleFunc("/chat/admins/search", chatHandler.SearchAdmins).Methods("GET", "OPTIONS")
	protected.HandleFunc("/chat/invitations", chatHandler.SendInvitation).Methods("POST", "OPTIONS")
	protected.HandleFunc("/chat/invitations", chatHandler.GetInvitations).Methods("GET", "OPTIONS")
	protected.HandleFunc("/chat/invitations/{id}/respond", chatHandler.RespondToInvitation).Methods("PUT", "OPTIONS")
	protected.HandleFunc("/chat/notifications/send", chatHandler.SendChatNotification).Methods("POST", "OPTIONS")
	
	// 👥 Routes Groupes de chat (protégées - accessible aux non-admins aussi)
	protected.HandleFunc("/chat/groups", chatGroupHandler.GetGroups).Methods("GET", "OPTIONS")
	protected.HandleFunc("/chat/groups/{group_id}/invite", chatGroupHandler.InviteToGroup).Methods("POST", "OPTIONS")
	protected.HandleFunc("/chat/groups/{group_id}/members", chatGroupHandler.GetGroupMembers).Methods("GET", "OPTIONS")
	protected.HandleFunc("/chat/groups/{group_id}/leave", chatGroupHandler.LeaveGroup).Methods("POST", "OPTIONS")
	protected.HandleFunc("/chat/groups/{group_id}/pending-invitations", chatGroupHandler.GetGroupPendingInvitations).Methods("GET", "OPTIONS")
	protected.HandleFunc("/chat/groups/{group_id}/invitations/pending", chatGroupHandler.GetGroupPendingInvitations).Methods("GET", "OPTIONS") // Alias
	protected.HandleFunc("/chat/groups/{group_id}/messages", chatGroupHandler.SendMessage).Methods("POST", "OPTIONS")
	protected.HandleFunc("/chat/groups/{group_id}/messages", chatGroupHandler.GetMessages).Methods("GET", "OPTIONS")
	protected.HandleFunc("/chat/groups/{group_id}/mark-read", chatGroupHandler.MarkAsRead).Methods("POST", "OPTIONS")
	protected.HandleFunc("/chat/group-invitations/pending", chatGroupHandler.GetPendingInvitations).Methods("GET", "OPTIONS")
	protected.HandleFunc("/chat/group-invitations/{invitation_id}/respond", chatGroupHandler.RespondToInvitation).Methods("PUT", "OPTIONS")

	// Routes médias (protégées - authentification requise)
	protected.HandleFunc("/evenements/{event_id}/medias", mediaHandler.CreateMedia).Methods("POST", "OPTIONS")
	protected.HandleFunc("/evenements/{event_id}/medias/{media_id}", mediaHandler.DeleteMedia).Methods("DELETE", "OPTIONS")

	// Routes admin inscriptions
	adminRouter.HandleFunc("/evenements/{event_id}/inscrits", inscriptionHandler.GetInscrits).Methods("GET", "OPTIONS")
	adminRouter.HandleFunc("/evenements/{event_id}/inscrits/{inscription_id}", inscriptionHandler.DeleteInscriptionAdmin).Methods("DELETE", "OPTIONS")
	adminRouter.HandleFunc("/evenements/{event_id}/inscrits/{inscription_id}/accompagnant/{index}", inscriptionHandler.DeleteAccompagnant).Methods("DELETE", "OPTIONS")

	// Créer un multiplexeur qui combine les deux routers
	mainHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Si c'est une requête WebSocket, utiliser rawRouter (sans middleware)
		if r.URL.Path == "/ws/chat" {
			rawRouter.ServeHTTP(w, r)
		} else {
			// Sinon, utiliser le router normal (avec middleware)
			router.ServeHTTP(w, r)
		}
	})

	// Démarrer le serveur
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mainHandler,
	}

	// Gérer l'arrêt gracieux du serveur
	go func() {
		log.Printf("🚀 Serveur démarré sur http://%s", addr)
		log.Printf("📝 Environnement: %s", cfg.Environment)
		log.Printf("🗄️  Base de données: MongoDB")
	log.Println("📋 Routes disponibles:")
	log.Println("   POST   /api/inscription                    - Inscription")
	log.Println("   POST   /api/connexion                      - Connexion")
	log.Println("   GET    /api/health                         - Health check")
	log.Println("   GET    /api/evenements/public              - Liste événements (public)")
	log.Println("   GET    /api/evenements/{id}                - Détails événement (public)")
	log.Println("   POST   /api/alerts/critical                - Alertes critiques admin (public)")
		log.Println("")
		log.Println("   🔔 Notifications VAPID (ancienne méthode):")
		log.Println("   GET    /api/notifications/vapid-public-key - Clé publique VAPID")
		log.Println("   POST   /api/notifications/subscribe        - S'abonner (VAPID)")
		log.Println("")
		log.Println("   🔥 Firebase Cloud Messaging (RECOMMANDÉ):")
		log.Println("   GET    /api/fcm/vapid-key                  - Clé VAPID Firebase")
		log.Println("   POST   /api/fcm/subscribe                  - S'abonner (FCM)")
		log.Println("")
		log.Println("   🔒 Routes protégées:")
		log.Println("   POST   /api/fcm/send                       - Envoyer à TOUS (FCM)")
		log.Println("   POST   /api/fcm/send-to-user               - Envoyer à un user (FCM)")
		log.Println("   GET    /api/protected/profile              - Profil utilisateur")
		log.Println("")
	log.Println("   👑 Routes Admin (admin=1 requis):")
	log.Println("   GET    /api/admin/utilisateurs             - Liste utilisateurs")
	log.Println("   PUT    /api/admin/utilisateurs/{id}        - Modifier utilisateur")
	log.Println("   DELETE /api/admin/utilisateurs/{id}        - Supprimer utilisateur")
	log.Println("   GET    /api/admin/evenements               - Liste événements")
	log.Println("   GET    /api/admin/evenements/{id}          - Détails événement")
	log.Println("   POST   /api/admin/evenements               - Créer événement")
	log.Println("   PUT    /api/admin/evenements/{id}          - Modifier événement")
	log.Println("   DELETE /api/admin/evenements/{id}          - Supprimer événement")
	log.Println("   GET    /api/admin/evenements/{id}/inscrits - Liste des inscrits")
	log.Println("   DELETE /api/admin/evenements/{id}/inscrits/{insc_id} - Supprimer inscription")
	log.Println("   DELETE /api/admin/evenements/{id}/inscrits/{insc_id}/accompagnant/{index} - Supprimer accompagnant")
	log.Println("   GET    /api/admin/stats                    - Statistiques globales")
	log.Println("   POST   /api/admin/notifications/send       - Envoyer notification admin")
	log.Println("   GET    /api/admin/codes-soiree             - Liste tous les codes")
	log.Println("   POST   /api/admin/code-soiree/generate     - Générer code soirée")
	log.Println("   GET    /api/admin/code-soiree/current      - Code soirée actuel")
	log.Println("")
	log.Println("   📝 Inscriptions aux événements (authentifié):")
	log.Println("   POST   /api/evenements/{id}/inscription    - S'inscrire")
	log.Println("   GET    /api/evenements/{id}/inscription    - Voir son inscription")
	log.Println("   PUT    /api/evenements/{id}/inscription    - Modifier inscription")
	log.Println("   DELETE /api/evenements/{id}/desinscription - Se désinscrire")
	log.Println("   GET    /api/mes-evenements                 - Mes événements inscrits")
	log.Println("")
	log.Println("   📸 Galerie médias :")
	log.Println("   GET    /api/evenements/{id}/medias         - Liste médias (public)")
	log.Println("   POST   /api/evenements/{id}/medias         - Ajouter média (authentifié)")
	log.Println("   DELETE /api/evenements/{id}/medias/{id}   - Supprimer média (authentifié)")
		log.Println("\n✨ Le serveur est prêt à recevoir des requêtes!")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Erreur du serveur: %v", err)
		}
	}()

	// Attendre le signal d'arrêt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\n🛑 Arrêt du serveur...")
	if err := server.Close(); err != nil {
		log.Printf("❌ Erreur lors de l'arrêt du serveur: %v", err)
	}
	log.Println("✓ Serveur arrêté proprement")
}
