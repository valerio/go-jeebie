# go-jeebie

[![CI](https://github.com/valerio/go-jeebie/workflows/CI/badge.svg)](https://github.com/valerio/go-jeebie/actions)

A Game Boy emulator written in Go.

## Requirements

- Go 1.23 or later

## Building and Running

```bash
# Build the emulator
make build

# Run with a Game Boy ROM
./bin/jeebie path/to/rom.gb

# Run tests
make test

# Run all tests, including snapshot tests for Blargg's test suite
make test-all
```


## Status

Still a work in progress. Currently passes all of Blargg's CPU instruction tests and Matt Currie's [dmg-acid2](https://github.com/mattcurrie/dmg-acid2) PPU test.

### Graphics Tests & Game Screenshots

dmg-acid2 test:

![dmg-acid2](screenshots/dmg-acid2.png)

Simple games running in the emulator:

![Tetris](screenshots/tetris.png) ![Super Mario Land](screenshots/super-mario-land.png)

### CPU Tests (Blargg's test suite - 11/11 passing)

![Blargg Tests](test/blargg/testdata/snapshots/01-special.png) ![Blargg Tests](test/blargg/testdata/snapshots/02-interrupts.png) ![Blargg Tests](test/blargg/testdata/snapshots/03-op%20sp,hl.png)

![Blargg Tests](test/blargg/testdata/snapshots/04-op%20r,imm.png) ![Blargg Tests](test/blargg/testdata/snapshots/05-op%20rp.png) ![Blargg Tests](test/blargg/testdata/snapshots/06-ld%20r,r.png)

![Blargg Tests](test/blargg/testdata/snapshots/07-jr,jp,call,ret,rst.png) ![Blargg Tests](test/blargg/testdata/snapshots/08-misc%20instrs.png) ![Blargg Tests](test/blargg/testdata/snapshots/09-op%20r,r.png)

![Blargg Tests](test/blargg/testdata/snapshots/10-bit%20ops.png) ![Blargg Tests](test/blargg/testdata/snapshots/11-op%20a,(hl).png)



## License

See the [license](./LICENSE.md) file for license rights and limitations (MIT).