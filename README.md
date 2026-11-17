# gymtracker

Personal workout tracker for two people. Single Go binary, no JavaScript
framework, no Node toolchain, no external Go modules.

## Run

```sh
go build -o gymtracker .
./gymtracker -addr 127.0.0.1:8080 -data /var/lib/gymtracker/data.json
```

All state lives in the one JSON file given by `-data` (created on first
write). Back it up by copying the file, or with the "export json" link in
the page header, which downloads the same document; a downloaded copy can be
passed straight back as `-data`.

## Expose it through Caddy

The app has profiles, not accounts. Put it behind Caddy with basic auth and
let Caddy handle TLS:

```caddyfile
gym.example.com {
	basic_auth {
		pedro  <hash from `caddy hash-password`>
		friend <hash from `caddy hash-password`>
	}
	reverse_proxy 127.0.0.1:8080
}
```

## Layout

- `model/`  domain types: programs, workouts, sessions, history
- `store/`  JSON file persistence with a mutex and atomic writes
- `web/`    `net/http` handlers, `html/template` pages and the stylesheet,
            all embedded into the binary

The only client side script is the rest timer on the session page. Every
form works with scripting disabled, and the progression chart on each lift
page is server rendered SVG.
