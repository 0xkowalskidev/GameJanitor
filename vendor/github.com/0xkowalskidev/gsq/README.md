# gsq

Query game servers from the command line or Go code. Supports Source engine (CS2, Rust, Ark, etc.) and Minecraft, with auto-detection and host scanning.

## Install

```bash
go install github.com/0xkowalskidev/gsq/cmd/gsq@latest
```

## Usage

```bash
gsq 192.168.1.100:27015                       # auto-detect protocol
gsq --game rust 192.168.1.100                  # specify game, use default port
gsq --players --game ark 192.168.1.100:27015   # include player list
gsq --json --game minecraft play.hypixel.net   # JSON output
gsq scan 192.168.1.100                         # find all game servers on a host
gsq scan --ports 25000-26000 192.168.1.100     # scan custom port range
gsq games                                      # list supported games
```

Output includes the address you queried, the inferred game port (what players connect to), and the query port (what the protocol responded on):

```
Name:        Rust Server
Address:     192.168.1.100
Game Port:   28015
Query Port:  28017
Game:        Rust
...
```

> **Note:** The game port is inferred from the query port using the known offset for each game (e.g. Rust query port = game port + 2). For servers with non-standard port layouts, such as containerized servers where port mappings don't preserve the offset, the displayed game port may be incorrect. The query port is always accurate.

## Library

```go
import "github.com/0xkowalskidev/gsq"

server, err := gsq.Query(ctx, "play.hypixel.net", 25565, gsq.QueryOptions{Game: "minecraft"})
server, err := gsq.Query(ctx, "192.168.1.100", 27015, gsq.QueryOptions{Game: "ark", Players: true})
servers, err := gsq.Discover(ctx, "192.168.1.100", gsq.DiscoverOptions{})
```

## EOS (Epic Online Services)

Some games (ARK: Survival Ascended, Palworld, Squad, The Isle: EVRIMA) use Epic's matchmaking API
instead of a direct query protocol. gsq ships with default credentials for these games,
but game updates may rotate them.

```bash
gsq --game asa 1.2.3.4:7777                    # uses built-in credentials
gsq --game asa --eos-client-id X --eos-client-secret Y 1.2.3.4:7777  # override credentials
```

If the built-in credentials stop working after a game update, you can extract current
credentials from the game's files or find them on community resources like opengsq.

> **Limitation:** The EOS matchmaking API returns at most 15 sessions per request with no
> pagination. If the server's IP hosts many game instances (common with large hosting
> providers), the target server may not appear in the results and the query will fail.

## License

MIT
