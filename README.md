

## Key Points / Notes
- [docker-compose.yml](docker-compose.yml) contains required services to setup local environment : Postgres, Redis
- [Makefile](Makefile) helper script
- [Cache client](external/redis/client.go) failure to intialize cache client will not stop application from running.
- For failed messages to send, we can maintain a seperate table for dead_letter_message which tracks
the msg ID and reason etc. Current implementation uses one table only.
- In [Message Domain Model](internal/messages/model.go), Status can be iota constant with `map[int]string` but required implementing marshalling methods, hence skipped in current implementation.
- [Work Pool Implementation](internal/messages/service.go) : Killing worker pool after every run (Implemented Approach) vs reusing worker pool 
    - Pros: 
        - Simple State Management: When the scheduler is paused, no background processes are running, making the "off" state very clean.
        - Resource Efficiency When Stopped: All goroutines and associated memory are freed completely when the work is done
    - Cons:
        - Performance Overhead: Constantly creating and destroying goroutines for every run introduces unnecessary performance overhead.