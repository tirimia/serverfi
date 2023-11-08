# Server-fi

![Epic logo comprised of a Gopher wearing a crown that resembles the Nix snowflake. It is also throwing tiny gophers that all carry tiny cardboard boxes to symbolize how it packages files in other go binaries](/assets/project_logo.png)

## What is this ?

I wanted to have an easy way to bundle and serve folders.

Serverfi helps by offering exactly that: a simple and convenient way to take a folder and get a binary that will serve the contents of said folder over HTTP.

### Why would you do it like this ?
Knew from the get-go I wanted to leverage the embed.FS feature of Golang. Thought to myself that I would need to bundle the Go compiler in my binary. Thank you, Nix, for helping me indulge in this horror.
