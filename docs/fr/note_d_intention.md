# SimuResto — Simulation multi-agents d'un restaurant universitaire

## Description du projet

**SimuResto** est un projet de simulation multi-agents en deux dimensions, développé en Go, visant à reproduire le fonctionnement d'un restaurant universitaire du CROUS. Les étudiants peuvent d'abord faire la queue à l'extérieur, puis suivre un parcours commun à l'intérieur : prendre un plateau, choisir une entrée, recevoir un plat principal servi par le personnel, payer, chercher une place pour manger et enfin rapporter leur plateau.

L'environnement comprend deux types d'agents : **les étudiants** et **le personnel du restaurant**. Les étudiants se déplacent et prennent des décisions autonomes, comme continuer à attendre ou quitter la file. Le personnel assure principalement le service des plats, le réapprovisionnement et l'encaissement. Les agents agissent selon leur état et les informations perçues dans l'environnement, avec certains comportements éventuellement probabilistes.

**Interactions entre agents :** Les étudiants peuvent s'influencer mutuellement en se regroupant, en faisant la queue ou en s'insérant dans une file. Ils échangent également des informations avec le personnel lors du choix des plats, du service et du paiement.

## Problématique

Comment l'affluence, le nombre de membres du personnel, la vitesse du service et les décisions individuelles des étudiants influencent-ils les temps d'attente, la congestion et le débit de service dans un restaurant universitaire ?

## Étapes de développement envisagées

1. **Prototype minimal :** Simuler un étudiant attendant qu'un membre du personnel lui serve un plat principal, puis effectuant le paiement, avec une visualisation en deux dimensions.
2. **Simulation cible :** Ajouter progressivement plusieurs étudiants, une file d'attente extérieure, les différentes étapes du service, le réapprovisionnement, la recherche de places et le départ après le repas.
3. **Extensions :** Introduire des facteurs supplémentaires, notamment la variation des arrivées selon les horaires, l'ajustement du nombre d'employés et les comportements sociaux tels que les arrivées en groupe ou les dépassements dans les files.

## Indicateurs clés

- **Temps d'attente :** Temps d'attente moyen et durée d'attente à chaque étape du service.
- **Congestion :** Longueur des files et nombre de personnes dans les différentes zones.
- **Débit de service :** Nombre d'étudiants servis par unité de temps.
- **Taux d'abandon :** Proportion d'étudiants quittant la file avant de prendre un plateau.
- **Taux d'occupation du personnel :** Proportion du temps consacrée aux tâches de service.

La comparaison de ces indicateurs dans différents scénarios permettra d'analyser l'évolution du fonctionnement global du restaurant.
