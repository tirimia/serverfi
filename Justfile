default:
    @just -l

test: test-build
    timeout 2 ./serverfi &
    @sleep 0.5
    curl 'http://localhost:8080'


test-build: build-flake
    ./result/bin/serverfi test/server-root

build-flake:
    nix build .#
