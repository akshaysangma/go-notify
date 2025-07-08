

## Notes
- [docker-compose.yml](docker-compose.yml) contains local infra related services : Postgres, Redis
- [Makefile](Makefile) helper script
- For failed messages to send, we can maintain a seperate table for dead_letter_message which tracks
the msg ID and reason etc. Currently, I have used one table only.

