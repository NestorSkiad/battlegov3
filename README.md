# battlegov3

A stateful REST API written in Golang that lets users play a game of Battleship. Users are not permanent, access is managed with tokens. System is distributed, with game sessions only existing on one node. Nodes communicate with each other to designate which node should load the game session in memory. A finger table exists in the database.

h3. Uses:
* Gin-gonic
* UUID
* PGX for PostgreSQL connection management

h3. Contribution guide:

Don't. I'm too busy and/or tired. Just fork it.

h3. License:

No for-profit use of this project, any original tech included herein or any of its derivatives without a written agreement with the author (Nestor Skiadas). Open-source, not-for-profit use allowed with attribution.

